package phpt

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

type Test struct {
	Name        string
	File        string
	SkipIf      string
	Expect      string
	ExpectF     string
	ExpectRegex string
	INI         string
	Clean       string
	Get         string
	Post        string
	PostRaw     string
	Env         string
	Cookie      string
	Headers     string
	Files       string
	ExitCode    int
	HasExitCode bool
}

type sapiContext struct {
	Get     []sapiEntry
	Post    []sapiEntry
	RawBody string
	RawType string
	INI     iniSettings
	Env     []sapiEntry
	Cookie  []sapiEntry
	Headers []sapiEntry
	Files   []sapiFile
	Temps   []string
}

type sapiEntry struct {
	Key   string
	Value string
}

type sapiFile struct {
	Field string
	Props []sapiEntry
}

type iniSettings struct {
	Values            map[string]string
	FileUploads       bool
	UploadMaxFilesize int64
	PostMaxSize       int64
	MaxFileUploads    int
	UploadTmpDir      string
}

type Result struct {
	Path       string
	Name       string
	Passed     bool
	Skipped    bool
	SkipReason string
	Output     string
	Expected   string
	Err        error
}

var sectionHeader = regexp.MustCompile(`^--([A-Z_]+)--\s*$`)

func Parse(input string) (*Test, error) {
	sections := map[string]string{}
	current := ""
	var b strings.Builder

	lines := strings.SplitAfter(normalizeLineEndings(input), "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n")
		if match := sectionHeader.FindStringSubmatch(trimmed); match != nil {
			if current != "" {
				sections[current] = trimSection(b.String())
				b.Reset()
			}
			current = match[1]
			continue
		}
		if current != "" {
			b.WriteString(line)
		}
	}
	if current != "" {
		sections[current] = trimSection(b.String())
	}

	test := &Test{
		Name:        strings.TrimSpace(sections["TEST"]),
		File:        sections["FILE"],
		SkipIf:      sections["SKIPIF"],
		Expect:      sections["EXPECT"],
		ExpectF:     sections["EXPECTF"],
		ExpectRegex: sections["EXPECTREGEX"],
		INI:         sections["INI"],
		Clean:       sections["CLEAN"],
		Get:         sections["GET"],
		Post:        sections["POST"],
		PostRaw:     sections["POST_RAW"],
		Env:         sections["ENV"],
		Cookie:      sections["COOKIE"],
		Headers:     sections["HEADERS"],
		Files:       sections["FILES"],
	}
	if exitCode, ok := sections["EXIT_CODE"]; ok && strings.TrimSpace(exitCode) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(exitCode))
		if err != nil {
			return nil, fmt.Errorf("invalid --EXIT_CODE-- value %q", strings.TrimSpace(exitCode))
		}
		test.ExitCode = parsed
		test.HasExitCode = true
	}
	if test.Name == "" {
		test.Name = "unnamed test"
	}
	if _, ok := sections["FILE"]; !ok {
		return nil, fmt.Errorf("missing --FILE-- section")
	}
	if _, ok := sections["EXPECT"]; !ok {
		if _, ok := sections["EXPECTF"]; !ok {
			if _, ok := sections["EXPECTREGEX"]; !ok {
				return nil, fmt.Errorf("missing --EXPECT--, --EXPECTF-- or --EXPECTREGEX-- section")
			}
		}
	}
	return test, nil
}

func Run(path string, input string) Result {
	test, err := Parse(input)
	if err != nil {
		return Result{Path: path, Name: filepath.Base(path), Err: err}
	}

	if test.SkipIf != "" {
		ctx, err := test.sapiContext()
		if err != nil {
			return Result{Path: path, Name: test.Name, Err: err}
		}
		output, err := runPHP(path, test.SkipIf, ctx)
		if err != nil {
			return Result{Path: path, Name: test.Name, Err: fmt.Errorf("SKIPIF error: %w", err)}
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(output)), "skip") {
			return Result{Path: path, Name: test.Name, Skipped: true, SkipReason: strings.TrimSpace(output)}
		}
	}

	ctx, ctxErr := test.sapiContext()
	if ctxErr != nil {
		return Result{Path: path, Name: test.Name, Err: ctxErr}
	}
	output, err := runPHP(path, test.File, ctx)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*runtime.ExitError); ok {
			exitCode = exitErr.Code
			err = nil
		} else {
			output += err.Error()
			exitCode = 1
		}
	}
	expected := test.Expect
	passed := false
	if test.ExpectRegex != "" {
		expected = test.ExpectRegex
		passed = expectRegexMatches(output, test.ExpectRegex)
	} else if test.ExpectF != "" {
		expected = test.ExpectF
		passed = expectFMatches(output, test.ExpectF)
	} else {
		passed = comparable(output) == comparable(test.Expect)
	}
	if test.HasExitCode && exitCode != test.ExitCode {
		passed = false
		if err == nil {
			err = fmt.Errorf("exit code mismatch: expected %d, got %d", test.ExitCode, exitCode)
		}
	}
	if test.Clean != "" {
		cleanCtx, cleanCtxErr := test.sapiContext()
		if cleanCtxErr != nil {
			passed = false
			if err == nil {
				err = cleanCtxErr
			}
		} else if _, cleanErr := runPHP(path, test.Clean, cleanCtx); cleanErr != nil {
			passed = false
			if err == nil {
				err = fmt.Errorf("CLEAN error: %w", cleanErr)
			}
		}
	}
	return Result{Path: path, Name: test.Name, Passed: passed, Output: output, Expected: expected, Err: err}
}

func (t *Test) sapiContext() (sapiContext, error) {
	rawHeaders, rawBody := parsePostRawSection(t.PostRaw)
	headers := append(parseHeadersSection(t.Headers), rawHeaders...)
	ini := parseINISection(t.INI)
	ctx := sapiContext{
		Get:     parseQuerySection(t.Get),
		Post:    parseQuerySection(t.Post),
		Env:     parseEnvSection(t.Env),
		Cookie:  parseCookieSection(t.Cookie),
		Headers: headers,
		Files:   parseFilesSection(t.Files),
		INI:     ini,
	}
	if rawBody == "" {
		return ctx, nil
	}
	ctx.RawBody = rawBody
	ctx.RawType = contentType(headers)
	if ini.PostMaxSize > 0 && int64(len(rawBody)) > ini.PostMaxSize {
		return ctx, nil
	}
	if isMultipart(ctx.RawType) {
		post, files, temps, err := parseMultipartPost(rawBody, ctx.RawType, ini)
		if err != nil {
			return ctx, err
		}
		ctx.Post = append(ctx.Post, post...)
		ctx.Files = append(ctx.Files, files...)
		ctx.Temps = append(ctx.Temps, temps...)
		return ctx, nil
	}
	if isURLEncoded(ctx.RawType) || ctx.RawType == "" {
		ctx.Post = append(ctx.Post, parseQuerySection(rawBody)...)
	}
	return ctx, nil
}

func runPHP(path string, source string, ctx sapiContext) (string, error) {
	for _, temp := range ctx.Temps {
		defer os.Remove(temp)
	}

	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors: %v", p.Errors())
	}

	interp := runtime.NewInterpreterWithBaseDir(filepath.Dir(path))
	interp.SetEntryFile(path)
	interp.SetUploadedFiles(ctx.Temps)
	installSAPIContext(interp, ctx, path)
	err := interp.Run(program)
	return interp.Out.String(), err
}

func installSAPIContext(interp *runtime.Interpreter, ctx sapiContext, path string) {
	get := arrayFromEntries(ctx.Get)
	post := arrayFromEntries(ctx.Post)
	env := arrayFromEntries(ctx.Env)
	cookie := arrayFromEntries(ctx.Cookie)
	files := filesArray(ctx.Files)
	request := runtime.NewArray()
	copyArrayEntries(request, get)
	copyArrayEntries(request, post)
	copyArrayEntries(request, cookie)

	server := runtime.NewArray()
	server.Set(runtime.StringKey("REQUEST_METHOD"), runtime.StringValue(requestMethod(ctx)))
	server.Set(runtime.StringKey("QUERY_STRING"), runtime.StringValue(encodeValues(ctx.Get)))
	server.Set(runtime.StringKey("REQUEST_URI"), runtime.StringValue(requestURI(ctx)))
	server.Set(runtime.StringKey("SCRIPT_FILENAME"), runtime.StringValue(path))
	server.Set(runtime.StringKey("SCRIPT_NAME"), runtime.StringValue("/"+filepath.Base(path)))
	server.Set(runtime.StringKey("PHP_SELF"), runtime.StringValue("/"+filepath.Base(path)))
	server.Set(runtime.StringKey("DOCUMENT_ROOT"), runtime.StringValue(filepath.Dir(path)))
	server.Set(runtime.StringKey("SERVER_NAME"), runtime.StringValue("localhost"))
	server.Set(runtime.StringKey("SERVER_PORT"), runtime.StringValue("80"))
	server.Set(runtime.StringKey("SERVER_PROTOCOL"), runtime.StringValue("HTTP/1.1"))
	if ctx.RawBody != "" {
		if ctx.RawType != "" {
			server.Set(runtime.StringKey("CONTENT_TYPE"), runtime.StringValue(ctx.RawType))
		}
		server.Set(runtime.StringKey("CONTENT_LENGTH"), runtime.StringValue(strconv.Itoa(len(ctx.RawBody))))
	} else if len(ctx.Post) != 0 {
		server.Set(runtime.StringKey("CONTENT_TYPE"), runtime.StringValue("application/x-www-form-urlencoded"))
		server.Set(runtime.StringKey("CONTENT_LENGTH"), runtime.StringValue(strconv.Itoa(len(encodeValues(ctx.Post)))))
	}
	if len(ctx.Cookie) != 0 {
		server.Set(runtime.StringKey("HTTP_COOKIE"), runtime.StringValue(cookieHeader(ctx.Cookie)))
	}
	installHeaders(server, ctx.Headers)

	interp.SetGlobal("_GET", runtime.ArrayValue(get))
	interp.SetGlobal("_POST", runtime.ArrayValue(post))
	interp.SetGlobal("_COOKIE", runtime.ArrayValue(cookie))
	interp.SetGlobal("_FILES", runtime.ArrayValue(files))
	interp.SetGlobal("_REQUEST", runtime.ArrayValue(request))
	interp.SetGlobal("_ENV", runtime.ArrayValue(env))
	interp.SetGlobal("_SERVER", runtime.ArrayValue(server))
}

func requestMethod(ctx sapiContext) string {
	if len(ctx.Post) != 0 || ctx.RawBody != "" || len(ctx.Files) != 0 {
		return "POST"
	}
	return "GET"
}

func requestURI(ctx sapiContext) string {
	query := encodeValues(ctx.Get)
	if query == "" {
		return "/"
	}
	return "/?" + query
}

func cookieHeader(entries []sapiEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Key+"="+entry.Value)
	}
	return strings.Join(parts, "; ")
}

func installHeaders(server *runtime.Array, headers []sapiEntry) {
	for _, header := range headers {
		key := serverHeaderKey(header.Key)
		if key == "" {
			continue
		}
		server.Set(runtime.StringKey(key), runtime.StringValue(header.Value))
	}
}

func contentType(headers []sapiEntry) string {
	for idx := len(headers) - 1; idx >= 0; idx-- {
		header := headers[idx]
		if strings.EqualFold(header.Key, "Content-Type") || strings.EqualFold(header.Key, "CONTENT_TYPE") {
			return header.Value
		}
	}
	return ""
}

func serverHeaderKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	upper := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	switch upper {
	case "CONTENT_TYPE", "CONTENT_LENGTH":
		return upper
	default:
		return "HTTP_" + upper
	}
}

func arrayFromEntries(values []sapiEntry) *runtime.Array {
	arr := runtime.NewArray()
	for _, entry := range values {
		arr.Set(runtime.StringKey(entry.Key), runtime.StringValue(entry.Value))
	}
	return arr
}

func filesArray(files []sapiFile) *runtime.Array {
	arr := runtime.NewArray()
	for _, file := range files {
		field, path := uploadFieldPath(file.Field)
		if field == "" {
			continue
		}
		if len(path) != 0 {
			fileArr, ok := existingFileArray(arr, field)
			if !ok {
				fileArr = newArrayUploadFile()
				arr.Set(runtime.StringKey(field), runtime.ArrayValue(fileArr))
			}
			appendArrayUploadFile(fileArr, path, file.Props)
			continue
		}
		fileArr := scalarUploadFile(file.Props)
		arr.Set(runtime.StringKey(field), runtime.ArrayValue(fileArr))
	}
	return arr
}

func scalarUploadFile(props []sapiEntry) *runtime.Array {
	fileArr := runtime.NewArray()
	for _, prop := range props {
		fileArr.Set(runtime.StringKey(prop.Key), fileValue(prop.Key, prop.Value))
	}
	return fileArr
}

func newArrayUploadFile() *runtime.Array {
	fileArr := runtime.NewArray()
	for _, key := range []string{"name", "type", "tmp_name", "error", "size"} {
		fileArr.Set(runtime.StringKey(key), runtime.ArrayValue(runtime.NewArray()))
	}
	return fileArr
}

func existingFileArray(files *runtime.Array, field string) (*runtime.Array, bool) {
	value, ok := files.Get(runtime.StringKey(field))
	if !ok || value.Kind != runtime.ArrayKind || value.A == nil {
		return nil, false
	}
	return value.A, true
}

func uploadFieldPath(field string) (string, []string) {
	start := strings.IndexByte(field, '[')
	if start < 0 {
		return field, nil
	}
	root := field[:start]
	var path []string
	rest := field[start:]
	for rest != "" {
		if !strings.HasPrefix(rest, "[") {
			return field, nil
		}
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return field, nil
		}
		path = append(path, rest[1:end])
		rest = rest[end+1:]
	}
	return root, path
}

func appendArrayUploadFile(fileArr *runtime.Array, path []string, props []sapiEntry) {
	for _, prop := range props {
		value, ok := fileArr.Get(runtime.StringKey(prop.Key))
		if !ok || value.Kind != runtime.ArrayKind || value.A == nil {
			value = runtime.ArrayValue(runtime.NewArray())
			fileArr.Set(runtime.StringKey(prop.Key), value)
		}
		setNestedUploadValue(value.A, path, fileValue(prop.Key, prop.Value))
	}
}

func setNestedUploadValue(arr *runtime.Array, path []string, value runtime.Value) {
	if len(path) == 0 {
		arr.Append(value)
		return
	}
	key := path[0]
	if len(path) == 1 {
		if key == "" {
			arr.Append(value)
			return
		}
		arr.Set(runtime.StringKey(key), value)
		return
	}
	var child *runtime.Array
	if key == "" {
		child = runtime.NewArray()
		arr.Append(runtime.ArrayValue(child))
	} else {
		current, ok := arr.Get(runtime.StringKey(key))
		if ok && current.Kind == runtime.ArrayKind && current.A != nil {
			child = current.A
		} else {
			child = runtime.NewArray()
			arr.Set(runtime.StringKey(key), runtime.ArrayValue(child))
		}
	}
	setNestedUploadValue(child, path[1:], value)
}

func fileValue(key string, value string) runtime.Value {
	switch key {
	case "size", "error":
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return runtime.IntValue(parsed)
		}
	}
	return runtime.StringValue(value)
}

func copyArrayEntries(dst *runtime.Array, src *runtime.Array) {
	for _, entry := range src.Entries {
		dst.Set(entry.Key, entry.Value)
	}
}

func parseQuerySection(section string) []sapiEntry {
	var out []sapiEntry
	section = strings.TrimSpace(section)
	if section == "" {
		return out
	}
	for _, part := range strings.Split(section, "&") {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			decodedKey = key
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			decodedValue = value
		}
		out = append(out, sapiEntry{Key: decodedKey, Value: decodedValue})
	}
	return out
}

func parseCookieSection(section string) []sapiEntry {
	var out []sapiEntry
	section = strings.TrimSpace(section)
	if section == "" {
		return out
	}
	section = strings.ReplaceAll(section, "\n", ";")
	for _, part := range strings.FieldsFunc(section, func(r rune) bool { return r == ';' || r == '&' }) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		decodedKey, err := url.QueryUnescape(strings.TrimSpace(key))
		if err != nil {
			decodedKey = strings.TrimSpace(key)
		}
		decodedValue, err := url.QueryUnescape(strings.TrimSpace(value))
		if err != nil {
			decodedValue = strings.TrimSpace(value)
		}
		out = append(out, sapiEntry{Key: decodedKey, Value: decodedValue})
	}
	return out
}

func parseHeadersSection(section string) []sapiEntry {
	var out []sapiEntry
	for _, line := range strings.Split(normalizeLineEndings(section), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			key, value, ok = strings.Cut(line, "=")
		}
		if !ok {
			continue
		}
		out = append(out, sapiEntry{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	return out
}

func parsePostRawSection(section string) ([]sapiEntry, string) {
	var headers []sapiEntry
	section = normalizeLineEndings(section)
	if strings.TrimSpace(section) == "" {
		return headers, ""
	}
	offset := 0
	for offset < len(section) {
		next := strings.IndexByte(section[offset:], '\n')
		lineEnd := len(section)
		lineNext := len(section)
		if next >= 0 {
			lineEnd = offset + next
			lineNext = lineEnd + 1
		}
		line := strings.TrimRight(section[offset:lineEnd], "\r")
		if strings.TrimSpace(line) == "" {
			return headers, section[lineNext:]
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return headers, section[offset:]
		}
		headers = append(headers, sapiEntry{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
		offset = lineNext
	}
	return headers, ""
}

func isMultipart(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		return strings.EqualFold(mediaType, "multipart/form-data")
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data")
}

func isURLEncoded(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		return strings.EqualFold(mediaType, "application/x-www-form-urlencoded")
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "application/x-www-form-urlencoded")
}

func parseMultipartPost(body string, contentType string, ini iniSettings) ([]sapiEntry, []sapiFile, []string, error) {
	boundary, err := multipartBoundary(contentType)
	if err != nil {
		return nil, nil, nil, err
	}
	reader := multipart.NewReader(strings.NewReader(body), boundary)
	var post []sapiEntry
	var files []sapiFile
	var temps []string
	formMaxFileSize := int64(-1)
	acceptedUploads := 0
	anonymousUploads := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanupTemps(temps)
			return nil, nil, nil, err
		}
		data, err := io.ReadAll(part)
		if err != nil {
			cleanupTemps(temps)
			return nil, nil, nil, err
		}
		name := part.FormName()
		filename, hasFilename := multipartFilename(part)
		if !hasFilename {
			if name == "" {
				continue
			}
			post = append(post, sapiEntry{Key: name, Value: string(data)})
			if name == "MAX_FILE_SIZE" {
				if parsed, ok := parseIntegerLimit(strings.TrimSpace(string(data))); ok {
					formMaxFileSize = parsed
				}
			}
			continue
		}
		if name == "" {
			name = strconv.Itoa(anonymousUploads)
			anonymousUploads++
		}
		if !ini.FileUploads {
			continue
		}
		if !validUploadFieldName(name) {
			continue
		}
		if filename == "" {
			files = append(files, uploadErrorFile(name, "", "", uploadErrNoFile, 0))
			continue
		}
		if ini.MaxFileUploads >= 0 && acceptedUploads >= ini.MaxFileUploads {
			continue
		}
		acceptedUploads++
		if ini.UploadMaxFilesize > 0 && int64(len(data)) > ini.UploadMaxFilesize {
			files = append(files, uploadErrorFile(name, filename, part.Header.Get("Content-Type"), uploadErrIniSize, int64(len(data))))
			continue
		}
		if formMaxFileSize > 0 && int64(len(data)) > formMaxFileSize {
			files = append(files, uploadErrorFile(name, filename, part.Header.Get("Content-Type"), uploadErrFormSize, int64(len(data))))
			continue
		}
		tmp, err := writeUploadedTemp(data, ini.UploadTmpDir)
		if err != nil {
			files = append(files, uploadErrorFile(name, filename, part.Header.Get("Content-Type"), uploadErrorCode(err), int64(len(data))))
			continue
		}
		temps = append(temps, tmp)
		files = append(files, sapiFile{Field: name, Props: []sapiEntry{
			{Key: "name", Value: filename},
			{Key: "type", Value: part.Header.Get("Content-Type")},
			{Key: "tmp_name", Value: tmp},
			{Key: "error", Value: "0"},
			{Key: "size", Value: strconv.Itoa(len(data))},
		}})
	}
	return post, files, temps, nil
}

const (
	uploadErrOK        = 0
	uploadErrIniSize   = 1
	uploadErrFormSize  = 2
	uploadErrNoFile    = 4
	uploadErrNoTmpDir  = 6
	uploadErrCantWrite = 7
)

func multipartFilename(part *multipart.Part) (string, bool) {
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return "", false
	}
	filename, ok := params["filename"]
	if !ok {
		return "", false
	}
	if filename == "" {
		return "", true
	}
	return filepath.Base(filename), true
}

func validUploadFieldName(name string) bool {
	depth := 0
	for idx := 0; idx < len(name); idx++ {
		switch name[idx] {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				return false
			}
			if idx+1 < len(name) && name[idx+1] != '[' {
				return false
			}
		}
	}
	return depth == 0
}

func uploadErrorFile(field string, filename string, contentType string, code int, size int64) sapiFile {
	if code != uploadErrOK {
		contentType = ""
	}
	return sapiFile{Field: field, Props: []sapiEntry{
		{Key: "name", Value: filename},
		{Key: "type", Value: contentType},
		{Key: "tmp_name", Value: ""},
		{Key: "error", Value: strconv.Itoa(code)},
		{Key: "size", Value: strconv.FormatInt(size, 10)},
	}}
}

func uploadErrorCode(err error) int {
	if os.IsNotExist(err) {
		return uploadErrNoTmpDir
	}
	return uploadErrCantWrite
}

func multipartBoundary(contentType string) (string, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err == nil && params["boundary"] != "" {
		return params["boundary"], nil
	}
	idx := strings.Index(strings.ToLower(contentType), "boundary=")
	if idx < 0 {
		return "", fmt.Errorf("missing multipart boundary")
	}
	boundary := strings.TrimSpace(contentType[idx+len("boundary="):])
	if boundary == "" {
		return "", fmt.Errorf("missing multipart boundary")
	}
	if boundary[0] == '"' {
		end := strings.IndexByte(boundary[1:], '"')
		if end < 0 {
			return "", fmt.Errorf("invalid multipart boundary")
		}
		return boundary[1 : 1+end], nil
	}
	if cut := strings.IndexAny(boundary, ",;"); cut >= 0 {
		boundary = boundary[:cut]
	}
	boundary = strings.TrimSpace(boundary)
	if boundary == "" {
		return "", fmt.Errorf("missing multipart boundary")
	}
	return boundary, nil
}

func writeUploadedTemp(data []byte, dir string) (string, error) {
	f, err := os.CreateTemp(dir, "php-src-go-upload-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func parseINISection(section string) iniSettings {
	values := map[string]string{}
	for _, line := range strings.Split(normalizeLineEndings(section), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		values[key] = value
	}
	return iniSettings{
		Values:            values,
		FileUploads:       iniBool(values, "file_uploads", true),
		UploadMaxFilesize: iniSize(values, "upload_max_filesize", 2*1024*1024),
		PostMaxSize:       iniSize(values, "post_max_size", 8*1024*1024),
		MaxFileUploads:    iniInt(values, "max_file_uploads", 20),
		UploadTmpDir:      values["upload_tmp_dir"],
	}
}

func iniBool(values map[string]string, key string, fallback bool) bool {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "on", "yes", "true":
		return true
	case "0", "off", "no", "false", "":
		return false
	default:
		return fallback
	}
}

func iniInt(values map[string]string, key string, fallback int) int {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, ok := parseIntegerLimit(value)
	if !ok {
		return fallback
	}
	return int(parsed)
}

func iniSize(values map[string]string, key string, fallback int64) int64 {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, ok := parseSizeLimit(value)
	if !ok {
		return fallback
	}
	return parsed
}

func parseIntegerLimit(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func parseSizeLimit(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	multiplier := int64(1)
	last := value[len(value)-1]
	switch last {
	case 'g', 'G':
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSpace(value[:len(value)-1])
	case 'm', 'M':
		multiplier = 1024 * 1024
		value = strings.TrimSpace(value[:len(value)-1])
	case 'k', 'K':
		multiplier = 1024
		value = strings.TrimSpace(value[:len(value)-1])
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed * multiplier, true
}

func cleanupTemps(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func parseFilesSection(section string) []sapiFile {
	files := map[string][]sapiEntry{}
	var order []string
	for _, line := range strings.Split(normalizeLineEndings(section), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		field, prop, ok := strings.Cut(strings.TrimSpace(key), ".")
		if !ok || field == "" || prop == "" {
			continue
		}
		if _, exists := files[field]; !exists {
			order = append(order, field)
		}
		files[field] = append(files[field], sapiEntry{Key: strings.TrimSpace(prop), Value: strings.TrimSpace(value)})
	}
	out := make([]sapiFile, 0, len(order))
	for _, field := range order {
		out = append(out, sapiFile{Field: field, Props: files[field]})
	}
	return out
}

func parseEnvSection(section string) []sapiEntry {
	var out []sapiEntry
	for _, line := range strings.Split(normalizeLineEndings(section), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			out = append(out, sapiEntry{Key: line})
			continue
		}
		out = append(out, sapiEntry{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	return out
}

func encodeValues(values []sapiEntry) string {
	query := url.Values{}
	for _, entry := range values {
		query.Set(entry.Key, entry.Value)
	}
	return query.Encode()
}

func comparable(s string) string {
	return strings.TrimRight(normalizeLineEndings(s), "\n")
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func trimSection(s string) string {
	s = normalizeLineEndings(s)
	return strings.TrimPrefix(strings.TrimRight(s, "\n"), "\n")
}

func expectFMatches(output string, pattern string) bool {
	re := regexp.QuoteMeta(comparable(pattern))
	re = strings.ReplaceAll(re, `%s`, `.*`)
	re = strings.ReplaceAll(re, `%a`, `.+`)
	re = strings.ReplaceAll(re, `%d`, `[+-]?\d+`)
	re = strings.ReplaceAll(re, `%i`, `[+-]?\d+`)
	re = strings.ReplaceAll(re, `%f`, `[+-]?(?:\d+\.\d+|\d+)`)
	re = strings.ReplaceAll(re, `%w`, `\s*`)
	re = strings.ReplaceAll(re, `%%`, `%`)
	matched, _ := regexp.MatchString(`(?s)^`+re+`$`, comparable(output))
	return matched
}

func expectRegexMatches(output string, pattern string) bool {
	matched, _ := regexp.MatchString(`(?s)^`+comparable(pattern)+`$`, comparable(output))
	return matched
}
