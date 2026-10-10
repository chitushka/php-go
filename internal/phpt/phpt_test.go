package phpt_test

import (
	"testing"

	"github.com/chitushka/php-go/internal/phpt"
)

func TestRunPassingPHPT(t *testing.T) {
	input := `--TEST--
basic arithmetic
--FILE--
<?php echo 1 + 2 * 3;
--EXPECT--
7
`
	result := phpt.Run("basic.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got err=%v output=%q expected=%q", result.Err, result.Output, result.Expected)
	}
}

func TestRunFailingPHPT(t *testing.T) {
	input := `--TEST--
failing arithmetic
--FILE--
<?php echo 1 + 1;
--EXPECT--
3
`
	result := phpt.Run("fail.phpt", input)
	if result.Passed {
		t.Fatal("expected failure")
	}
	if result.Output != "2" {
		t.Fatalf("got output %q", result.Output)
	}
}

func TestParseRequiresFileAndExpect(t *testing.T) {
	if _, err := phpt.Parse("--TEST--\nmissing\n"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRunExpectF(t *testing.T) {
	input := `--TEST--
expectf wildcard
--FILE--
<?php echo 'value 123';
--EXPECTF--
value %d
`
	result := phpt.Run("expectf.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunSkipIf(t *testing.T) {
	input := `--TEST--
skip example
--SKIPIF--
<?php echo 'skip unsupported feature';
--FILE--
<?php echo 'no';
--EXPECT--
no
`
	result := phpt.Run("skip.phpt", input)
	if !result.Skipped {
		t.Fatalf("expected skip, got %+v", result)
	}
}

func TestRunExpectedRuntimeError(t *testing.T) {
	input := `--TEST--
expected runtime error
--FILE--
<?php class Sample { private $a = 1; } $s = new Sample(); echo $s->a;
--EXPECTF--
cannot access private property Sample::$a
`
	result := phpt.Run("error.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunExpectRegex(t *testing.T) {
	input := `--TEST--
expect regex
--FILE--
<?php echo 'user-42';
--EXPECTREGEX--
user-\d+
`
	result := phpt.Run("regex.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunExitCode(t *testing.T) {
	input := `--TEST--
expected failure exit code
--FILE--
<?php class Sample { private $a = 1; } $s = new Sample(); echo $s->a;
--EXPECTF--
cannot access private property Sample::$a
--EXIT_CODE--
1
`
	result := phpt.Run("exit.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunClean(t *testing.T) {
	input := `--TEST--
clean section
--FILE--
<?php echo 'ok';
--CLEAN--
<?php $x = 1;
--EXPECT--
ok
`
	result := phpt.Run("clean.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunSAPISections(t *testing.T) {
	input := `--TEST--
sapi sections
--GET--
name=php&version=5.3
--POST--
action=run
--ENV--
APP_ENV=test
--FILE--
<?php
echo $_GET['name'];
echo ':';
echo $_GET['version'];
echo ':';
echo $_POST['action'];
echo ':';
echo $_REQUEST['name'];
echo ':';
echo $_REQUEST['action'];
echo ':';
echo $_ENV['APP_ENV'];
echo ':';
echo $_SERVER['REQUEST_METHOD'];
--EXPECT--
php:5.3:run:php:run:test:POST
`
	result := phpt.Run("sapi.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunGetPreservesOrder(t *testing.T) {
	input := `--TEST--
get order
--GET--
a=1&b=2&c=3
--FILE--
<?php foreach ($_GET as $k => $v) { echo $k; echo $v; }
--EXPECT--
a1b2c3
`
	result := phpt.Run("get_order.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunExtendedSAPISections(t *testing.T) {
	input := `--TEST--
extended sapi sections
--GET--
page=1
--POST--
action=save
--COOKIE--
sid=abc; theme=dark
--HEADERS--
Host: example.test
User-Agent: php-src-go
Content-Type: text/plain
--FILE--
<?php
echo $_COOKIE['sid'];
echo ':';
echo $_COOKIE['theme'];
echo ':';
echo $_REQUEST['sid'];
echo ':';
echo $_SERVER['HTTP_HOST'];
echo ':';
echo $_SERVER['HTTP_USER_AGENT'];
echo ':';
echo $_SERVER['CONTENT_TYPE'];
echo ':';
echo $_SERVER['HTTP_COOKIE'];
echo ':';
echo $_SERVER['REQUEST_URI'];
echo ':';
echo count($_FILES);
--EXPECT--
abc:dark:abc:example.test:php-src-go:text/plain:sid=abc; theme=dark:/?page=1:0
`
	result := phpt.Run("extended_sapi.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunFilesSection(t *testing.T) {
	input := `--TEST--
files section
--FILES--
avatar.name=hello.txt
avatar.type=text/plain
avatar.tmp_name=/tmp/php123
avatar.error=0
avatar.size=5
--FILE--
<?php
echo $_FILES['avatar']['name'];
echo ':';
echo $_FILES['avatar']['type'];
echo ':';
echo $_FILES['avatar']['tmp_name'];
echo ':';
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['size'];
--EXPECT--
hello.txt:text/plain:/tmp/php123:0:5
`
	result := phpt.Run("files.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunPostRawMultipart(t *testing.T) {
	input := `--TEST--
post raw multipart
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="title"

hello
--php-src-go
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

abcde
--php-src-go--
--FILE--
<?php
echo $_SERVER['REQUEST_METHOD'];
echo ':';
echo $_SERVER['CONTENT_TYPE'];
echo ':';
echo $_POST['title'];
echo ':';
echo $_FILES['avatar']['name'];
echo ':';
echo $_FILES['avatar']['type'];
echo ':';
echo is_file($_FILES['avatar']['tmp_name']);
echo ':';
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['size'];
--EXPECT--
POST:multipart/form-data; boundary=php-src-go:hello:hello.txt:text/plain:1:0:5
`
	result := phpt.Run("post_raw_multipart.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunPostRawNestedMultipart(t *testing.T) {
	input := `--TEST--
nested post raw multipart
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="gallery[group][]"; filename="a.txt"
Content-Type: text/plain

a
--php-src-go
Content-Disposition: form-data; name="gallery[group][]"; filename="b.txt"
Content-Type: text/plain

bb
--php-src-go
Content-Disposition: form-data; name="gallery[avatar][front]"; filename="front.txt"
Content-Type: text/plain

front
--php-src-go--
--FILE--
<?php
echo $_FILES['gallery']['name']['group'][0];
echo ':';
echo $_FILES['gallery']['size']['group'][0];
echo ':';
echo $_FILES['gallery']['name']['group'][1];
echo ':';
echo $_FILES['gallery']['size']['group'][1];
echo ':';
echo $_FILES['gallery']['name']['avatar']['front'];
echo ':';
echo $_FILES['gallery']['size']['avatar']['front'];
--EXPECT--
a.txt:1:b.txt:2:front.txt:5
`
	result := phpt.Run("nested_multipart.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunPostMaxSizeSkipsRequestBody(t *testing.T) {
	input := `--TEST--
post max size
--INI--
post_max_size=8
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="title"

hello
--php-src-go--
--FILE--
<?php
echo $_SERVER['REQUEST_METHOD'];
echo ':';
echo $_SERVER['CONTENT_LENGTH'];
echo ':';
echo count($_POST);
echo ':';
echo count($_FILES);
--EXPECT--
POST:79:0:0
`
	result := phpt.Run("post_max_size.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunUploadLimitsAndRFC1867Edges(t *testing.T) {
	input := `--TEST--
upload limits and rfc1867 edges
--INI--
upload_max_filesize=3
max_file_uploads=3
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="MAX_FILE_SIZE"

2
--php-src-go
Content-Disposition: form-data; name="ok"; filename="ok.txt"
Content-Type: text/plain

aa
--php-src-go
Content-Disposition: form-data; name="formbig"; filename="form.txt"
Content-Type: text/plain

aaa
--php-src-go
Content-Disposition: form-data; name="inibig"; filename="ini.txt"
Content-Type: text/plain

aaaa
--php-src-go
Content-Disposition: form-data; name="empty"; filename=""
Content-Type: text/plain

--php-src-go
Content-Disposition: form-data; name="ignored"; filename="ignored.txt"
Content-Type: text/plain

x
--php-src-go--
--FILE--
<?php
echo $_FILES['ok']['size'];
echo ':';
echo $_FILES['ok']['error'];
echo ':';
echo is_uploaded_file($_FILES['ok']['tmp_name']);
echo ':';
echo $_FILES['formbig']['error'];
echo ':';
echo $_FILES['formbig']['tmp_name'];
echo ':';
echo $_FILES['formbig']['size'];
echo ':';
echo $_FILES['inibig']['error'];
echo ':';
echo $_FILES['inibig']['size'];
echo ':';
echo $_FILES['empty']['error'];
echo ':';
echo $_FILES['empty']['name'];
echo ':';
echo $_FILES['empty']['size'];
echo ':';
echo count($_FILES);
--EXPECT--
2:0:1:2::3:1:4:4::0:4
`
	result := phpt.Run("upload_limits.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunFileUploadsOffKeepsPostFields(t *testing.T) {
	input := `--TEST--
file uploads off
--INI--
file_uploads=0
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="title"

hello
--php-src-go
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

abc
--php-src-go--
--FILE--
<?php
echo $_POST['title'];
echo ':';
echo count($_FILES);
--EXPECT--
hello:0
`
	result := phpt.Run("file_uploads_off.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunUploadTmpDirFailure(t *testing.T) {
	input := `--TEST--
upload tmp dir failure
--INI--
upload_tmp_dir=/php-src-go-missing-upload-tmp
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

abc
--php-src-go--
--FILE--
<?php
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['tmp_name'];
echo ':';
echo is_uploaded_file($_FILES['avatar']['tmp_name']);
--EXPECT--
6::
`
	result := phpt.Run("upload_tmp_dir_failure.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunZeroUploadSizeLimitsAreUnlimited(t *testing.T) {
	input := `--TEST--
zero upload size limits
--INI--
upload_max_filesize=0
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="MAX_FILE_SIZE"

0
--php-src-go
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

abcd
--php-src-go--
--FILE--
<?php
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['size'];
echo ':';
echo is_uploaded_file($_FILES['avatar']['tmp_name']);
--EXPECT--
0:4:1
`
	result := phpt.Run("zero_upload_size_limits.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunRFC1867AnonymousAndInvalidUploadNames(t *testing.T) {
	input := `--TEST--
rfc1867 anonymous and invalid upload names
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="bad[a]x"; filename="bad.txt"
Content-Type: text/plain

bad
--php-src-go
Content-Disposition: form-data; filename="anon.txt"
Content-Type: text/plain

z
--php-src-go--
--FILE--
<?php
echo count($_FILES);
echo ':';
echo $_FILES['0']['name'];
echo ':';
echo $_FILES['0']['size'];
--EXPECT--
1:anon.txt:1
`
	result := phpt.Run("rfc1867_names.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}
