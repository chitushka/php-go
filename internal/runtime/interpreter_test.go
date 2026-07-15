package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

func run(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreter()
	if err := i.Run(program); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return i.Out.String()
}

func runErr(t *testing.T, src string) error {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreter()
	return i.Run(program)
}

func parseErrors(src string) []string {
	p := parser.New(lexer.New(src))
	p.ParseProgram()
	return p.Errors()
}

func TestEchoArithmetic(t *testing.T) {
	if got := run(t, "<?php echo 1 + 2 * 3;"); got != "7" {
		t.Fatalf("got %q", got)
	}
}
func TestVariablesAndIf(t *testing.T) {
	if got := run(t, "<?php $a = 4; if ($a > 2) { echo 'yes'; } else { echo 'no'; }"); got != "yes" {
		t.Fatalf("got %q", got)
	}
}
func TestWhile(t *testing.T) {
	if got := run(t, "<?php $i = 0; while ($i < 3) { echo $i; $i = $i + 1; }"); got != "012" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayForeachValues(t *testing.T) {
	got := run(t, "<?php $items = array('a', 'b', 'c'); foreach ($items as $v) { echo $v; }")
	if got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayForeachKeysAndValues(t *testing.T) {
	got := run(t, "<?php $items = array('x' => 7, 3 => 'z', 'last'); foreach ($items as $k => $v) { echo $k; echo ':'; echo $v; echo ';'; }")
	if got != "x:7;3:z;4:last;" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionReturn(t *testing.T) {
	got := run(t, "<?php function add($a, $b) { return $a + $b; } echo add(2, 5);")
	if got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionLocalScope(t *testing.T) {
	got := run(t, "<?php $x = 1; function setx() { $x = 9; return $x; } echo setx(); echo $x;")
	if got != "91" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionImplicitNullReturn(t *testing.T) {
	got := run(t, "<?php function noop() { $x = 1; } echo 'a'; echo noop(); echo 'b';")
	if got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionWithForeach(t *testing.T) {
	src := "<?php function sum_values($items) { $sum = 0; foreach ($items as $v) { $sum = $sum + $v; } return $sum; } echo sum_values(array(1, 2, 3));"
	got := run(t, src)
	if got != "6" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionCanBeCalledBeforeDeclaration(t *testing.T) {
	got := run(t, "<?php echo add(2, 3); function add($a, $b) { return $a + $b; }")
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func runFile(t *testing.T, baseDir, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreterWithBaseDir(baseDir)
	if err := i.Run(program); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return i.Out.String()
}

func TestIncludeSharesCurrentScope(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vars.php"), []byte("<?php $x = $x + 2; echo $x;"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php $x = 3; include 'vars.php'; echo $x;")
	if got != "55" {
		t.Fatalf("got %q", got)
	}
}

func TestRequireDefinesFunctions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "math.php"), []byte("<?php function inc($v) { return $v + 1; }"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php require 'math.php'; echo inc(4);")
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func TestNestedIncludeUsesIncludedFileDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "lib")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "value.php"), []byte("<?php $value = 8;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "loader.php"), []byte("<?php include 'value.php';"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php include 'lib/loader.php'; echo $value;")
	if got != "8" {
		t.Fatalf("got %q", got)
	}
}

func TestIncludeOnceAndRequireOnceShareIncludedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "once.php"), []byte("<?php echo 'I'; $count = $count + 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php $count = 0; include_once 'once.php'; include_once './once.php'; require_once 'once.php'; echo ':'; echo $count;")
	if got != "I:1" {
		t.Fatalf("got %q", got)
	}
}

func TestClassConstructorMethodAndThis(t *testing.T) {
	src := "<?php class Box { function __construct($value) { $this->value = $value; } function get() { return $this->value; } } $box = new Box(9); echo $box->get();"
	got := run(t, src)
	if got != "9" {
		t.Fatalf("got %q", got)
	}
}

func TestOldStyleConstructorMethodAndThis(t *testing.T) {
	src := "<?php class Box { function Box($value) { $this->value = $value; } function get() { return $this->value; } } $box = new Box(9); echo $box->get();"
	got := run(t, src)
	if got != "9" {
		t.Fatalf("got %q", got)
	}
}

func TestInheritedOldStyleConstructor(t *testing.T) {
	src := "<?php class Base { function Base($value) { $this->value = $value; } } class Child extends Base { function get() { return $this->value; } } $child = new Child(7); echo $child->get();"
	got := run(t, src)
	if got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestConstructPreferredOverOldStyleConstructor(t *testing.T) {
	src := "<?php class Box { function Box($value) { $this->value = 'old'; } function __construct($value) { $this->value = $value; } function get() { return $this->value; } } $box = new Box('new'); echo $box->get();"
	got := run(t, src)
	if got != "new" {
		t.Fatalf("got %q", got)
	}
}

func TestDestructorRunsAtShutdown(t *testing.T) {
	src := "<?php class Box { function __destruct() { echo ' destruct'; } } $box = new Box(); echo 'body';"
	got := run(t, src)
	if got != "body destruct" {
		t.Fatalf("got %q", got)
	}
}

func TestDestructorsRunInReverseCreationOrder(t *testing.T) {
	src := "<?php class Item { function __construct($name) { $this->name = $name; } function __destruct() { echo $this->name; } } $a = new Item('a'); $b = new Item('b'); echo 'body';"
	got := run(t, src)
	if got != "bodyba" {
		t.Fatalf("got %q", got)
	}
}

func TestInheritedDestructorRunsAtShutdown(t *testing.T) {
	src := "<?php class Base { function __destruct() { echo 'base'; } } class Child extends Base {} $child = new Child(); echo 'body';"
	got := run(t, src)
	if got != "bodybase" {
		t.Fatalf("got %q", got)
	}
}

func TestDestructorRunsAfterExit(t *testing.T) {
	p := parser.New(lexer.New("<?php class Box { function __destruct() { echo ' destruct'; } } $box = new Box(); echo 'body'; exit(' exit');"))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreter()
	err := i.Run(program)
	if exitErr, ok := err.(*runtime.ExitError); !ok || exitErr.Code != 0 {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := i.Out.String(); got != "body exit destruct" {
		t.Fatalf("got %q", got)
	}
}

func TestIncludedObjectDestructorRunsAtMainShutdown(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class Included { function __destruct() { echo ' destruct'; } } $included = new Included(); echo 'include';"
	if err := os.WriteFile(filepath.Join(dir, "included.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php include 'included.php'; echo ' main';")
	if got != "include main destruct" {
		t.Fatalf("got %q", got)
	}
}

func TestObjectPropertyAssignmentOutsideClass(t *testing.T) {
	src := "<?php class Box { function get() { return $this->value; } } $box = new Box(); $box->value = 4; echo $box->value; echo $box->get();"
	got := run(t, src)
	if got != "44" {
		t.Fatalf("got %q", got)
	}
}

func TestClassCanBeUsedBeforeDeclaration(t *testing.T) {
	src := "<?php $calc = new Calc(); echo $calc->add(2, 3); class Calc { function add($a, $b) { return $a + $b; } }"
	got := run(t, src)
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func TestDynamicClassNameInNew(t *testing.T) {
	src := "<?php class Box { function __construct($value) { $this->value = $value; } function get() { return $this->value; } } $class = 'Box'; $box = new $class(8); echo $box->get();"
	got := run(t, src)
	if got != "8" {
		t.Fatalf("got %q", got)
	}
}

func TestVariableVariableClassNameInNew(t *testing.T) {
	src := "<?php class Box { function get() { return 'box'; } } $name = 'class'; $class = 'Box'; $box = new $$name(); echo $box->get();"
	got := run(t, src)
	if got != "box" {
		t.Fatalf("got %q", got)
	}
}

func TestClassNamesAreCaseInsensitive(t *testing.T) {
	src := "<?php class BaseCase { const KIND = 'base'; public static $count = 1; public function value() { return 'v'; } public static function bump() { self::$count = self::$count + 1; return self::$count; } } class ChildCase extends basecase {} $name = 'childcase'; $child = new $name(); echo $child instanceof CHILDCASE; echo ':'; echo childcase::KIND; echo ':'; echo CHILDCASE::BUMP(); echo ':'; echo $child instanceof BASECASE; echo ':'; echo $child->VALUE();"
	got := run(t, src)
	if got != "1:base:2:1:v" {
		t.Fatalf("got %q", got)
	}
}

func TestInterfaceNamesAreCaseInsensitive(t *testing.T) {
	src := "<?php interface NamedCase { const LABEL = 'named'; public function name(); } interface LabeledCase extends namedcase {} class UserCase implements labeledcase { public function NAME() { return 'Ada'; } } $user = new USERCASE(); echo $user instanceof NAMEDCASE; echo ':'; echo labeledcase::LABEL; echo ':'; echo USERCASE::LABEL; echo ':'; echo $user->nAmE();"
	got := run(t, src)
	if got != "1:named:named:Ada" {
		t.Fatalf("got %q", got)
	}
}

func TestClassLikeRedeclareIsCaseInsensitive(t *testing.T) {
	err := runErr(t, "<?php class Box {} class box {}")
	if err == nil || err.Error() != "cannot redeclare class box" {
		t.Fatalf("unexpected error: %v", err)
	}
	err = runErr(t, "<?php interface Named {} interface named {}")
	if err == nil || err.Error() != "cannot redeclare interface named" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncludedClass(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class Doubler { function run($value) { return $value * 2; } }"
	if err := os.WriteFile(filepath.Join(dir, "doubler.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php require 'doubler.php'; $d = new Doubler(); echo $d->run(6);")
	if got != "12" {
		t.Fatalf("got %q", got)
	}
}

func TestAutoloadLoadsClassOnNew(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class Autoloaded { public function name() { return 'loaded'; } }"
	if err := os.WriteFile(filepath.Join(dir, "autoloaded.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'autoloaded.php'; } $obj = new Autoloaded(); echo $obj->name();")
	if got != "loaded" {
		t.Fatalf("got %q", got)
	}
}

func TestAutoloadLoadsClassForStaticAccess(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class AutoloadedStatic { const LABEL = 'static'; public static $count = 2; public static function name() { return self::LABEL; } }"
	if err := os.WriteFile(filepath.Join(dir, "autoloaded_static.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'autoloaded_static.php'; } echo autoloadedstatic::LABEL; echo ':'; echo AUTOLOADEDSTATIC::$count; echo ':'; echo AutoloadedStatic::NAME();")
	if got != "static:2:static" {
		t.Fatalf("got %q", got)
	}
}

func TestAutoloadLoadsParentClassDuringDeclaration(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class AutoloadedParent { public function name() { return 'parent'; } }"
	if err := os.WriteFile(filepath.Join(dir, "autoloaded_parent.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'autoloaded_parent.php'; } class AutoloadedChild extends AutoloadedParent {} $obj = new AutoloadedChild(); echo $obj->name();")
	if got != "parent" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPIntrospectionFunctions(t *testing.T) {
	dir := t.TempDir()
	src := "<?php interface InspectContract { public function run(); } class InspectBase { private $hidden = 1; protected function hiddenMethod() {} } class InspectTarget extends InspectBase implements InspectContract { public $name = 'Ada'; private function secret() {} public function run() { return get_class(); } }"
	if err := os.WriteFile(filepath.Join(dir, "introspection.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'introspection.php'; } echo class_exists('InspectTarget', false) ? 'y' : 'n'; echo ':'; echo interface_exists('InspectContract') ? 'y' : 'n'; echo ':'; echo class_exists('inspecttarget') ? 'y' : 'n'; $obj = new inspecttarget(); $obj->dynamic = 1; echo ':'; echo get_class($obj); echo ':'; echo $obj->run(); echo ':'; echo method_exists($obj, 'secret') ? 'y' : 'n'; echo ':'; echo method_exists('InspectBase', 'hiddenMethod') ? 'y' : 'n'; echo ':'; echo property_exists($obj, 'hidden') ? 'y' : 'n'; echo ':'; echo property_exists($obj, 'dynamic') ? 'y' : 'n'; echo ':'; echo property_exists('InspectTarget', 'missing') ? 'y' : 'n';")
	if got != "n:y:y:InspectTarget:InspectTarget:y:y:y:y:n" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPHierarchyIntrospectionFunctions(t *testing.T) {
	dir := t.TempDir()
	src := "<?php interface ParentContract {} interface ChildContract extends ParentContract {} class HierBase {} class HierChild extends HierBase implements ChildContract { public function parentName() { return get_parent_class(); } }"
	if err := os.WriteFile(filepath.Join(dir, "hierarchy.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'hierarchy.php'; } $obj = new HierChild(); echo get_parent_class($obj); echo ':'; echo $obj->parentName(); echo ':'; echo is_subclass_of($obj, 'HierBase') ? 'y' : 'n'; echo ':'; echo is_subclass_of('HierChild', 'ParentContract') ? 'y' : 'n'; echo ':'; echo is_a($obj, 'HierChild') ? 'y' : 'n'; echo ':'; echo is_a('HierChild', 'HierBase') ? 'y' : 'n'; echo ':'; echo is_a('HierChild', 'HierBase', true) ? 'y' : 'n';")
	if got != "HierBase:HierBase:y:y:y:n:y" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPHierarchyArrayIntrospectionFunctions(t *testing.T) {
	dir := t.TempDir()
	src := "<?php interface ParentContract {} interface ChildContract extends ParentContract {} class HierBase {} class HierChild extends HierBase implements ChildContract {}"
	if err := os.WriteFile(filepath.Join(dir, "hierarchy_arrays.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'hierarchy_arrays.php'; } echo class_parents('HierChild', false) ? 'loaded' : 'missing'; echo ':'; $parents = class_parents('hierchild'); echo $parents['HierBase']; echo ':'; $interfaces = class_implements('HierChild'); echo $interfaces['ChildContract']; echo ':'; echo $interfaces['ParentContract']; echo ':'; $ifaceParents = class_implements('ChildContract'); echo $ifaceParents['ParentContract'];")
	if got != "missing:HierBase:ChildContract:ParentContract:ParentContract" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPListingIntrospectionFunctions(t *testing.T) {
	src := "<?php class ListBase { public $base = 'base'; protected $hidden = 'hidden'; public static $root = 'root'; protected static $hiddenRoot = 'hidden-root'; public function baseMethod() {} protected function hiddenMethod() {} } class ListTarget extends ListBase { public $name = 'Ada'; private $secret = 's'; public static $kind = 'user'; private static $privateKind = 'secret-kind'; public function run() {} private function secretMethod() {} public function inside() { $vars = get_object_vars($this); $classVars = get_class_vars('ListTarget'); $methods = get_class_methods('ListTarget'); return $vars['secret'] . ':' . $classVars['secret'] . ':' . (in_array('secretMethod', $methods) ? 'secret' : 'no'); } } $obj = new ListTarget(); $obj->dynamic = 'dyn'; $vars = get_object_vars($obj); echo $vars['base']; echo ':'; echo $vars['name']; echo ':'; echo $vars['dynamic']; echo ':'; echo array_key_exists('secret', $vars) ? 'secret' : 'no'; $classVars = get_class_vars('listtarget'); echo ':'; echo $classVars['base']; echo ':'; echo $classVars['name']; echo ':'; echo $classVars['kind']; echo ':'; echo array_key_exists('privateKind', $classVars) ? 'private' : 'no'; $methods = get_class_methods($obj); echo ':'; echo in_array('run', $methods) ? 'run' : 'no'; echo ':'; echo in_array('secretMethod', $methods) ? 'secret' : 'no'; echo ':'; echo $obj->inside();"
	got := run(t, src)
	if got != "base:Ada:dyn:no:base:Ada:user:no:run:no:s:s:secret" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPDeclaredClassInterfaceIntrospectionFunctions(t *testing.T) {
	src := "<?php interface DeclaredContract {} interface DeclaredChild extends DeclaredContract {} class DeclaredBase {} class DeclaredUser extends DeclaredBase implements DeclaredChild {} $classes = get_declared_classes(); $interfaces = get_declared_interfaces(); echo in_array('Exception', $classes) ? 'exception' : 'no'; echo ':'; echo in_array('DeclaredBase', $classes) ? 'base' : 'no'; echo ':'; echo in_array('DeclaredUser', $classes) ? 'user' : 'no'; echo ':'; echo in_array('DeclaredContract', $interfaces) ? 'contract' : 'no'; echo ':'; echo in_array('DeclaredChild', $interfaces) ? 'child' : 'no';"
	got := run(t, src)
	if got != "exception:base:user:contract:child" {
		t.Fatalf("got %q", got)
	}
}

func TestOOPDeclaredIntrospectionIncludesAutoloadedTypes(t *testing.T) {
	dir := t.TempDir()
	src := "<?php interface DeclaredAutoloadContract {} class DeclaredAutoloaded implements DeclaredAutoloadContract {}"
	if err := os.WriteFile(filepath.Join(dir, "declared_autoload.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'declared_autoload.php'; } $classes = get_declared_classes(); $interfaces = get_declared_interfaces(); echo in_array('DeclaredAutoloaded', $classes) ? 'loaded' : 'missing'; echo ':'; echo in_array('DeclaredAutoloadContract', $interfaces) ? 'loaded' : 'missing'; echo ':'; echo class_exists('DeclaredAutoloaded') ? 'class' : 'no'; $classes = get_declared_classes(); $interfaces = get_declared_interfaces(); echo ':'; echo in_array('DeclaredAutoloaded', $classes) ? 'loaded' : 'missing'; echo ':'; echo in_array('DeclaredAutoloadContract', $interfaces) ? 'loaded' : 'missing';")
	if got != "missing:missing:class:loaded:loaded" {
		t.Fatalf("got %q", got)
	}
}

func TestClassAliasAutoloadAndLookup(t *testing.T) {
	dir := t.TempDir()
	src := "<?php class AliasSource { const LABEL = 'source'; public static $count = 2; public static function bump() { self::$count = self::$count + 1; return self::$count; } public function name() { return 'object'; } }"
	if err := os.WriteFile(filepath.Join(dir, "alias_source.php"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php function __autoload($class) { include 'alias_source.php'; } echo class_alias('AliasSource', 'AliasTarget') ? 'alias' : 'no'; echo ':'; echo class_exists('aliastarget', false) ? 'exists' : 'missing'; echo ':'; $obj = new AliasTarget(); echo get_class($obj); echo ':'; echo $obj->name(); echo ':'; echo AliasTarget::LABEL; echo ':'; echo AliasTarget::bump(); echo ':'; echo is_a($obj, 'AliasTarget') ? 'isa' : 'no'; echo ':'; $classes = get_declared_classes(); echo in_array('AliasTarget', $classes) ? 'declared' : 'missing'; echo ':'; echo class_alias('AliasSource', 'AliasTarget') ? 'again' : 'duplicate';")
	if got != "alias:exists:AliasSource:object:source:3:isa:declared:duplicate" {
		t.Fatalf("got %q", got)
	}
}

func TestIsObjectBuiltinForOOPValues(t *testing.T) {
	src := "<?php class ObjectType {} class_alias('ObjectType', 'ObjectAlias'); $obj = new ObjectType(); $alias = new ObjectAlias(); $fn = function() {}; echo is_object($obj) ? 'object' : 'no'; echo ':'; echo is_object($alias) ? 'alias' : 'no'; echo ':'; echo is_object($fn) ? 'closure' : 'no'; echo ':'; echo is_object(array()) ? 'array' : 'no'; echo ':'; echo is_object(null) ? 'null' : 'no';"
	got := run(t, src)
	if got != "object:alias:closure:no:no" {
		t.Fatalf("got %q", got)
	}
}

func TestIsCallableBuiltinForOOPValues(t *testing.T) {
	src := "<?php function callable_fn() {} class CallableTarget { public function run() {} private function hidden() {} public static function stat() {} } class CallableMagic { public function __call($name, $args) {} public static function __callStatic($name, $args) {} } class_alias('CallableTarget', 'CallableAlias'); $obj = new CallableTarget(); $magic = new CallableMagic(); $fn = function() {}; echo is_callable('callable_fn') ? 'fn' : 'no'; echo ':'; echo is_callable($fn) ? 'closure' : 'no'; echo ':'; echo is_callable(array($obj, 'run')) ? 'method' : 'no'; echo ':'; echo is_callable(array($obj, 'hidden')) ? 'hidden' : 'no'; echo ':'; echo is_callable(array($magic, 'missing')) ? 'magic' : 'no'; echo ':'; echo is_callable(array('CallableTarget', 'stat')) ? 'static' : 'no'; echo ':'; echo is_callable('CallableAlias::stat') ? 'alias' : 'no'; echo ':'; echo is_callable(array('CallableMagic', 'missing')) ? 'static-magic' : 'no'; echo ':'; echo is_callable(array('Missing', 'missing'), true) ? 'syntax' : 'no'; echo ':'; echo is_callable(array($obj, '__clone')) ? 'clone' : 'no';"
	got := run(t, src)
	if got != "fn:closure:method:no:magic:static:alias:static-magic:syntax:no" {
		t.Fatalf("got %q", got)
	}
}

func TestThrowAndCatchException(t *testing.T) {
	src := "<?php try { throw new Exception('boom'); } catch (Exception $e) { echo $e->getMessage(); } echo ' done';"
	got := run(t, src)
	if got != "boom done" {
		t.Fatalf("got %q", got)
	}
}

func TestExceptionAccessors(t *testing.T) {
	src := "<?php try { throw new Exception('boom', 7); } catch (Exception $e) { echo $e->getMessage(); echo ':'; echo $e->getCode(); echo ':'; echo $e->getFile(); echo ':'; echo $e->getLine(); echo ':'; echo $e; }"
	got := run(t, src)
	if got != "boom:7::1:Exception: boom" {
		t.Fatalf("got %q", got)
	}
}

func TestExceptionPreviousAndTraceAsString(t *testing.T) {
	src := "<?php function wrap() { fail(); } function fail() { throw new Exception('outer', 2, new Exception('inner', 1)); } try { wrap(); } catch (Exception $e) { $previous = $e->getPrevious(); echo $previous->getMessage(); echo ':'; echo $e->getTraceAsString(); }"
	got := run(t, src)
	if got != "inner:#0 fail()\n#1 wrap()" {
		t.Fatalf("got %q", got)
	}
}

func TestExceptionTraceArray(t *testing.T) {
	src := "<?php class Runner { public function wrap() { fail(); } } function fail() { throw new Exception('outer'); } try { $runner = new Runner(); $runner->wrap(); } catch (Exception $e) { $trace = $e->getTrace(); echo $trace[0]['function']; echo ':'; echo $trace[1]['class']; echo ':'; echo $trace[1]['type']; echo ':'; echo $trace[1]['function']; }"
	got := run(t, src)
	if got != "fail:Runner::::wrap" {
		t.Fatalf("got %q", got)
	}
}

func TestMultipleCatchChoosesFirstMatchingHandler(t *testing.T) {
	src := "<?php class First extends Exception {} class Second extends Exception {} try { throw new Second('two'); } catch (First $e) { echo 'first'; } catch (Second $e) { echo $e->getMessage(); } catch (Exception $e) { echo 'base'; }"
	got := run(t, src)
	if got != "two" {
		t.Fatalf("got %q", got)
	}
}

func TestExceptionPropagatesFromFunction(t *testing.T) {
	src := "<?php function fail() { throw new Exception('bad'); } try { fail(); echo 'no'; } catch (Exception $e) { echo $e->getMessage(); }"
	got := run(t, src)
	if got != "bad" {
		t.Fatalf("got %q", got)
	}
}

func TestExceptionPropagatesFromMethod(t *testing.T) {
	src := "<?php class Failer { function run() { throw new Exception('method'); } } try { $f = new Failer(); $f->run(); } catch (Exception $e) { echo $e->getMessage(); }"
	got := run(t, src)
	if got != "method" {
		t.Fatalf("got %q", got)
	}
}

func TestUncaughtExceptionReturnsRuntimeError(t *testing.T) {
	p := parser.New(lexer.New("<?php throw new Exception('uncaught');"))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreter()
	err := i.Run(program)
	if err == nil || err.Error() != "uncaught Exception: uncaught" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClassExtendsInheritedMethodAndProperty(t *testing.T) {
	src := "<?php class Base { public $value = 3; public function get() { return $this->value; } } class Child extends Base { public function double() { return $this->get() * 2; } } $c = new Child(); echo $c->double();"
	got := run(t, src)
	if got != "6" {
		t.Fatalf("got %q", got)
	}
}

func TestStaticPropertyAndMethod(t *testing.T) {
	src := "<?php class Counter { public static $count = 0; public static function inc() { Counter::$count = Counter::$count + 1; return Counter::$count; } } echo Counter::inc(); echo Counter::inc();"
	got := run(t, src)
	if got != "12" {
		t.Fatalf("got %q", got)
	}
}

func TestVisibilityModifiersAreAccepted(t *testing.T) {
	src := "<?php class Sample { private $a = 2; protected function getA() { return $this->a; } public function expose() { return $this->getA(); } } $s = new Sample(); echo $s->expose();"
	got := run(t, src)
	if got != "2" {
		t.Fatalf("got %q", got)
	}
}

func TestPrivatePropertyAccessDeniedOutsideClass(t *testing.T) {
	err := runErr(t, "<?php class Sample { private $a = 2; } $s = new Sample(); echo $s->a;")
	if err == nil || err.Error() != "cannot access private property Sample::$a" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProtectedMethodAccessDeniedOutsideHierarchy(t *testing.T) {
	err := runErr(t, "<?php class Sample { protected function hidden() { return 1; } } $s = new Sample(); echo $s->hidden();")
	if err == nil || err.Error() != "cannot access protected method Sample::hidden" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProtectedAccessAllowedFromChild(t *testing.T) {
	src := "<?php class Base { protected $value = 4; protected function getValue() { return $this->value; } } class Child extends Base { public function expose() { return $this->getValue(); } } $c = new Child(); echo $c->expose();"
	got := run(t, src)
	if got != "4" {
		t.Fatalf("got %q", got)
	}
}

func TestPrivateStaticPropertyAccessDeniedOutsideClass(t *testing.T) {
	err := runErr(t, "<?php class Sample { private static $count = 1; } echo Sample::$count;")
	if err == nil || err.Error() != "cannot access private static property Sample::$count" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanWidenInheritedPropertyVisibility(t *testing.T) {
	src := "<?php class Base { protected $value = 1; } class Child extends Base { public $value = 2; } $c = new Child(); echo $c->value;"
	got := run(t, src)
	if got != "2" {
		t.Fatalf("got %q", got)
	}
}

func TestCannotReduceInheritedPropertyVisibility(t *testing.T) {
	err := runErr(t, "<?php class Base { public $value = 1; } class Child extends Base { protected $value = 2; }")
	if err == nil || err.Error() != "cannot reduce visibility of inherited property Base::$value in Child" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCannotChangeInheritedPropertyStaticModifier(t *testing.T) {
	err := runErr(t, "<?php class Base { public $value = 1; } class Child extends Base { public static $value = 2; }")
	if err == nil || err.Error() != "cannot change static modifier of inherited property Base::$value in Child" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanRedeclarePrivateParentProperty(t *testing.T) {
	src := "<?php class Base { private $value = 1; } class Child extends Base { public $value = 2; } $c = new Child(); echo $c->value;"
	got := run(t, src)
	if got != "2" {
		t.Fatalf("got %q", got)
	}
}

func TestSelfStaticAccess(t *testing.T) {
	src := "<?php class Counter { private static $count = 0; public static function inc() { self::$count = self::$count + 1; return self::$count; } } echo Counter::inc(); echo Counter::inc();"
	got := run(t, src)
	if got != "12" {
		t.Fatalf("got %q", got)
	}
}

func TestLateStaticAccess(t *testing.T) {
	src := "<?php class Base { public static $name = 'base'; public static function name() { return static::$name; } } class Child extends Base { public static $name = 'child'; } echo Child::name();"
	got := run(t, src)
	if got != "child" {
		t.Fatalf("got %q", got)
	}
}

func TestGetCalledClass(t *testing.T) {
	src := "<?php class Base { public static function who() { return get_called_class(); } public function instanceWho() { return get_called_class(); } } class Child extends Base {} $c = new Child(); echo Base::who(); echo ':'; echo Child::who(); echo ':'; echo $c->instanceWho();"
	got := run(t, src)
	if got != "Base:Child:Child" {
		t.Fatalf("got %q", got)
	}
}

func TestParentMethodCall(t *testing.T) {
	src := "<?php class Base { protected function value() { return 4; } } class Child extends Base { public function value2() { return parent::value() * 2; } } $c = new Child(); echo $c->value2();"
	got := run(t, src)
	if got != "8" {
		t.Fatalf("got %q", got)
	}
}

func TestCannotExtendFinalClass(t *testing.T) {
	err := runErr(t, "<?php final class Base {} class Child extends Base {}")
	if err == nil || err.Error() != "class Child cannot extend final class Base" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCannotOverrideFinalMethod(t *testing.T) {
	err := runErr(t, "<?php class Base { final public function value() { return 1; } } class Child extends Base { public function value() { return 2; } }")
	if err == nil || err.Error() != "cannot override final method Base::value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCannotReduceInheritedMethodVisibility(t *testing.T) {
	err := runErr(t, "<?php class Base { public function value() { return 1; } } class Child extends Base { protected function value() { return 2; } }")
	if err == nil || err.Error() != "cannot reduce visibility of inherited method Base::value in Child" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanWidenInheritedMethodVisibility(t *testing.T) {
	src := "<?php class Base { protected function value() { return 1; } } class Child extends Base { public function value() { return 2; } } $c = new Child(); echo $c->value();"
	got := run(t, src)
	if got != "2" {
		t.Fatalf("got %q", got)
	}
}

func TestCannotChangeInheritedMethodStaticModifier(t *testing.T) {
	err := runErr(t, "<?php class Base { public static function value() { return 1; } } class Child extends Base { public function value() { return 2; } }")
	if err == nil || err.Error() != "cannot change static modifier of inherited method Base::value in Child" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCanAddOptionalInheritedMethodParameter(t *testing.T) {
	src := "<?php class Base { public function value($name) { return $name; } } class Child extends Base { public function value($name, $suffix = '!') { return $name . $suffix; } } $c = new Child(); echo $c->value('go'); echo $c->value('go', '?');"
	got := run(t, src)
	if got != "go!go?" {
		t.Fatalf("got %q", got)
	}
}

func TestCannotAddRequiredInheritedMethodParameter(t *testing.T) {
	err := runErr(t, "<?php class Base { public function value($name) { return $name; } } class Child extends Base { public function value($name, $suffix) { return $name . $suffix; } }")
	if err == nil || err.Error() != "method Child::value cannot require 2 parameters to match Base::value requiring 1" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCannotRemoveInheritedMethodParameter(t *testing.T) {
	err := runErr(t, "<?php class Base { public function value($name, $suffix = '!') { return $name . $suffix; } } class Child extends Base { public function value($name) { return $name; } }")
	if err == nil || err.Error() != "method Child::value must accept at least 2 parameters to match Base::value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInheritedMethodParameterReferenceModifierMustMatch(t *testing.T) {
	err := runErr(t, "<?php class Base { public function value(&$name) { return $name; } } class Child extends Base { public function value($name) { return $name; } }")
	if err == nil || err.Error() != "method Child::value parameter 1 reference modifier must match Base::value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInheritedMethodReturnByReferenceModifierMustMatch(t *testing.T) {
	err := runErr(t, "<?php class Base { public function &value() { return $this->value; } } class Child extends Base { public function value() { return 1; } }")
	if err == nil || err.Error() != "method Child::value return-by-reference modifier must match Base::value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAbstractClassCannotBeInstantiated(t *testing.T) {
	err := runErr(t, "<?php abstract class Base { abstract public function value(); } $b = new Base();")
	if err == nil || err.Error() != "cannot instantiate abstract class Base" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConcreteClassMustImplementAbstractMethods(t *testing.T) {
	err := runErr(t, "<?php abstract class Base { abstract public function value(); } class Child extends Base {}")
	if err == nil || err.Error() != "class Child contains abstract methods and must be abstract" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInterfaceImplementation(t *testing.T) {
	src := "<?php interface Named { public function name(); } class User implements Named { public function name() { return 'Ada'; } } $u = new User(); echo $u->name();"
	got := run(t, src)
	if got != "Ada" {
		t.Fatalf("got %q", got)
	}
}

func TestInterfaceImplementationCanAddOptionalParameter(t *testing.T) {
	src := "<?php interface Named { public function name($first); } class User implements Named { public function name($first, $last = '') { return $first . $last; } } $u = new User(); echo $u->name('Ada'); echo $u->name('Ada', ' Lovelace');"
	got := run(t, src)
	if got != "AdaAda Lovelace" {
		t.Fatalf("got %q", got)
	}
}

func TestInterfaceExtendsInterface(t *testing.T) {
	src := "<?php interface A { public function a(); } interface B extends A { public function b(); } class C implements B { public function a() { return 'a'; } public function b() { return 'b'; } } $c = new C(); echo $c->a(); echo $c->b();"
	got := run(t, src)
	if got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestClassMustImplementInterfaceMethod(t *testing.T) {
	err := runErr(t, "<?php interface Named { public function name(); } class User implements Named {}")
	if err == nil || err.Error() != "class User must implement interface method Named::name" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInterfaceMethodMustBePublic(t *testing.T) {
	err := runErr(t, "<?php interface Named { public function name(); } class User implements Named { protected function name() { return 'x'; } }")
	if err == nil || err.Error() != "class User interface method name must be public" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClassConstants(t *testing.T) {
	src := "<?php class Config { const NAME = 'php'; const VERSION = 53; public function name() { return self::NAME; } } $c = new Config(); echo Config::VERSION; echo ':'; echo $c->name();"
	got := run(t, src)
	if got != "53:php" {
		t.Fatalf("got %q", got)
	}
}

func TestInheritedClassConstantsAndParentAccess(t *testing.T) {
	src := "<?php class Base { const VALUE = 4; } class Child extends Base { const LOCAL = 2; public function total() { return parent::VALUE + self::LOCAL; } } $c = new Child(); echo Child::VALUE; echo ':'; echo $c->total();"
	got := run(t, src)
	if got != "4:6" {
		t.Fatalf("got %q", got)
	}
}

func TestInterfaceConstants(t *testing.T) {
	src := "<?php interface Named { const KIND = 'iface'; } echo Named::KIND;"
	got := run(t, src)
	if got != "iface" {
		t.Fatalf("got %q", got)
	}
}

func TestInheritedInterfaceConstants(t *testing.T) {
	src := "<?php interface A { const VALUE = 4; } interface B extends A { const LOCAL = 2; } class C implements B {} echo B::VALUE; echo ':'; echo C::LOCAL; echo ':'; echo C::VALUE;"
	got := run(t, src)
	if got != "4:2:4" {
		t.Fatalf("got %q", got)
	}
}

func TestInstanceOfClassAndParent(t *testing.T) {
	src := "<?php class Base {} class Child extends Base {} $c = new Child(); echo $c instanceof Child; echo $c instanceof Base; echo $c instanceof Missing;"
	got := run(t, src)
	if got != "11" {
		t.Fatalf("got %q", got)
	}
}

func TestInstanceOfInterface(t *testing.T) {
	src := "<?php interface Named {} interface UserNamed extends Named {} class User implements UserNamed {} $u = new User(); echo $u instanceof UserNamed; echo $u instanceof Named;"
	got := run(t, src)
	if got != "11" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicGetAndSet(t *testing.T) {
	src := "<?php class Bag { private $value = 'old'; public function __set($name, $value) { $this->value = $value; } public function __get($name) { return $this->value; } } $b = new Bag(); $b->item = 'new'; echo $b->item;"
	got := run(t, src)
	if got != "new" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicIssetAndEmpty(t *testing.T) {
	src := "<?php class Bag { private $items; public function __construct() { $this->items = array('yes' => 'value', 'zero' => '0'); } public function __isset($name) { return isset($this->items[$name]); } public function __get($name) { return $this->items[$name]; } } $b = new Bag(); if (isset($b->yes)) { echo 'y'; } if (!isset($b->missing)) { echo 'm'; } if (empty($b->zero)) { echo 'z'; } if (!empty($b->yes)) { echo 'v'; }"
	got := run(t, src)
	if got != "ymzv" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicUnset(t *testing.T) {
	src := "<?php class Bag { private $items; public function __construct() { $this->items = array('x' => 1); } public function __isset($name) { return isset($this->items[$name]); } public function __unset($name) { unset($this->items[$name]); echo 'u:' . $name; } } $b = new Bag(); unset($b->x); if (!isset($b->x)) { echo ':gone'; }"
	got := run(t, src)
	if got != "u:x:gone" {
		t.Fatalf("got %q", got)
	}
}

func TestUnsetConstructTargets(t *testing.T) {
	src := "<?php class UnsetBase { public static $base = 'base'; } class UnsetTarget extends UnsetBase { public static $count = 7; public $name = 'Ada'; } $var = 'v'; $items = array('x' => 1, 'y' => 2); $obj = new UnsetTarget(); $prop = 'name'; $static = 'slot'; $slot = 'count'; unset($var, $items['x'], $obj->$prop, UnsetTarget::$$static); echo isset($var) ? 'var' : 'no-var'; echo ':'; echo isset($items['x']) ? 'x' : 'no-x'; echo ':'; echo isset($obj->name) ? 'name' : 'no-name'; echo ':'; echo isset(UnsetTarget::$count) ? 'count' : 'no-count'; unset(UnsetTarget::$base); echo ':'; echo isset(UnsetBase::$base) ? 'base' : 'no-base';"
	got := run(t, src)
	if got != "no-var:no-x:no-name:no-count:no-base" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCall(t *testing.T) {
	src := "<?php class Dynamic { public function __call($name, $args) { echo $name; foreach ($args as $arg) { echo ':'; echo $arg; } } } $d = new Dynamic(); $d->missing(1, 'x');"
	got := run(t, src)
	if got != "missing:1:x" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCallForInaccessibleMethod(t *testing.T) {
	src := "<?php class Dynamic { private function hidden() { return 'no'; } public function __call($name, $args) { return $name; } } $d = new Dynamic(); echo $d->hidden();"
	got := run(t, src)
	if got != "hidden" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCallStatic(t *testing.T) {
	src := "<?php class Dynamic { public static function __callStatic($name, $args) { echo $name; foreach ($args as $arg) { echo ':'; echo $arg; } } } Dynamic::missing(1, 'x');"
	got := run(t, src)
	if got != "missing:1:x" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCallStaticForInaccessibleMethod(t *testing.T) {
	src := "<?php class Dynamic { private static function hidden() { return 'no'; } public static function __callStatic($name, $args) { return $name; } } echo Dynamic::hidden();"
	got := run(t, src)
	if got != "hidden" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCallStaticForDynamicMethodName(t *testing.T) {
	src := "<?php class Dynamic { public static function __callStatic($name, $args) { return $name . ':' . $args[0]; } } $method = 'missing'; echo Dynamic::$method('x');"
	got := run(t, src)
	if got != "missing:x" {
		t.Fatalf("got %q", got)
	}
}

func TestCloneCopiesObjectProperties(t *testing.T) {
	src := "<?php class Box { public $value = 1; } $a = new Box(); $b = clone $a; $b->value = 2; echo $a->value; echo ':'; echo $b->value;"
	got := run(t, src)
	if got != "1:2" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicCloneRunsOnClonedObject(t *testing.T) {
	src := "<?php class Box { public $value = 'old'; public function __clone() { $this->value = 'clone'; echo 'c'; } } $a = new Box(); $b = clone $a; echo ':'; echo $a->value; echo ':'; echo $b->value;"
	got := run(t, src)
	if got != "c:old:clone" {
		t.Fatalf("got %q", got)
	}
}

func TestPrivateCloneDeniedOutsideClass(t *testing.T) {
	err := runErr(t, "<?php class Box { private function __clone() {} } $a = new Box(); $b = clone $a;")
	if err == nil || err.Error() != "cannot access private method Box::__clone" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrivateCloneAllowedInsideClass(t *testing.T) {
	src := "<?php class Box { public $value = 'old'; private function __clone() { $this->value = 'clone'; } public function copy() { return clone $this; } } $a = new Box(); $b = $a->copy(); echo $a->value; echo ':'; echo $b->value;"
	got := run(t, src)
	if got != "old:clone" {
		t.Fatalf("got %q", got)
	}
}

func TestCloneCannotBeCalledAsMethod(t *testing.T) {
	err := runErr(t, "<?php class Box { public function __clone() {} } $a = new Box(); $a->__clone();")
	if err == nil || err.Error() != "cannot call __clone directly; use clone" {
		t.Fatalf("unexpected error: %v", err)
	}
	err = runErr(t, "<?php class Box { public function __call($name, $args) { echo 'call'; } } $a = new Box(); $method = '__clone'; $a->$method();")
	if err == nil || err.Error() != "cannot call __clone directly; use clone" {
		t.Fatalf("unexpected dynamic error: %v", err)
	}
}

func TestCloneRequiresObject(t *testing.T) {
	err := runErr(t, "<?php $x = clone 1;")
	if err == nil || err.Error() != "clone expects object" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMagicSleepSelectsSerializedProperties(t *testing.T) {
	src := "<?php class User { public $name = 'Ada'; public $secret = 'hidden'; public function __sleep() { return array('name'); } } $u = new User(); echo serialize($u);"
	got := run(t, src)
	if got != `O:4:"User":1:{s:4:"name";s:3:"Ada";}` {
		t.Fatalf("got %q", got)
	}
}

func TestMagicWakeupRunsAfterUnserialize(t *testing.T) {
	src := `<?php class User { public $name; public $ready = 'no'; public function __wakeup() { $this->ready = 'yes'; } } $u = unserialize('O:4:"User":1:{s:4:"name";s:3:"Ada";}'); echo $u->name; echo ':'; echo $u->ready;`
	got := run(t, src)
	if got != "Ada:yes" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicToString(t *testing.T) {
	src := "<?php class Label { public function __toString() { return 'label'; } } $label = new Label(); echo $label;"
	got := run(t, src)
	if got != "label" {
		t.Fatalf("got %q", got)
	}
}

func TestMagicToStringRequiresStringReturn(t *testing.T) {
	err := runErr(t, "<?php class Label { public function __toString() { return 123; } } $label = new Label(); echo $label;")
	if err == nil || err.Error() != "__toString must return string" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConcatAndMagicToString(t *testing.T) {
	src := "<?php class Label { public function __toString() { return 'B'; } } $label = new Label(); echo 'A' . $label . 3;"
	got := run(t, src)
	if got != "AB3" {
		t.Fatalf("got %q", got)
	}
}

func TestLogicalAndOrShortCircuit(t *testing.T) {
	src := "<?php $x = 0; function fail() { return 1 / 0; } if ($x && fail()) { echo 'bad'; } else { echo 'and'; } if (1 || fail()) { echo 'or'; }"
	got := run(t, src)
	if got != "andor" {
		t.Fatalf("got %q", got)
	}
}

func TestP0OperatorsWordBitwiseShiftTernary(t *testing.T) {
	src := "<?php function fail() { return 1 / 0; } if (0 and fail()) { echo 'bad'; } else { echo 'and'; } if (1 or fail()) { echo 'or'; } echo ':'; echo 6 & 3; echo ':'; echo 4 | 1; echo ':'; echo 5 ^ 3; echo ':'; echo ~0; echo ':'; echo 1 << 3; echo ':'; echo 8 >> 1; echo ':'; echo +5; echo ':'; echo 0 ? 'bad' : 'ternary'; echo ':'; echo 'fallback' ?: 'bad'; echo ':'; if (1 xor 0) { echo 'xor'; }"
	got := run(t, src)
	if got != "andor:2:5:6:-1:8:4:5:ternary:fallback:xor" {
		t.Fatalf("got %q", got)
	}
}

func TestWordLogicalOperatorsHaveLowerPrecedenceThanAssignment(t *testing.T) {
	src := "<?php $b = true or false; echo $b ? '1' : '0'; echo ':'; $b = false or true; echo $b ? '1' : '0'; echo ':'; $b = true || false; echo $b ? '1' : '0'; echo ':'; $b = false || true; echo $b ? '1' : '0'; echo ':'; $b = true and false; echo $b ? '1' : '0'; echo ':'; $b = true && false; echo $b ? '1' : '0';"
	got := run(t, src)
	if got != "1:0:1:1:1:0" {
		t.Fatalf("got %q", got)
	}
}

func TestP0Casts(t *testing.T) {
	src := "<?php echo (int) '12px'; echo ':'; echo (integer) true; echo ':'; echo (float) '1.5x'; echo ':'; echo (string) 7; echo ':'; echo (bool) ''; echo ':'; echo (boolean) 'x'; echo ':'; $a = (array) 'v'; echo count($a); echo ':'; echo $a[0]; echo ':'; $empty = (array) null; echo count($empty); echo ':'; $o = (object) array('a' => 9); echo $o->a; echo ':'; $s = (object) 3; echo $s->scalar; echo ':'; echo (unset) 'x'; echo 'done';"
	got := run(t, src)
	if got != "12:1:1.5:7::1:1:v:0:9:3:done" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayIndexFetchAndAssign(t *testing.T) {
	got := run(t, "<?php $items = array('a' => 1, 2 => 'b'); echo $items['a']; echo $items[2]; $items['a'] = 7; echo $items['a'];")
	if got != "1b7" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayAppendAssign(t *testing.T) {
	src := "<?php $items = array('first'); $items[] = 'second'; $items[5] = 'sixth'; $items[] = 'seventh'; echo count($items); echo ':'; echo $items[0]; echo ':'; echo $items[1]; echo ':'; echo $items[6]; $new[] = 'created'; echo ':'; echo $new[0]; $null = null; $null[] = 'from-null'; echo ':'; echo $null[0];"
	got := run(t, src)
	if got != "4:first:second:seventh:created:from-null" {
		t.Fatalf("got %q", got)
	}
}

func TestStringIndexFetch(t *testing.T) {
	got := run(t, "<?php $s = 'php'; echo $s[0]; echo $s[1]; echo $s[2];")
	if got != "php" {
		t.Fatalf("got %q", got)
	}
}

func TestForLoopBreakAndContinue(t *testing.T) {
	src := "<?php for ($i = 0; $i < 6; $i = $i + 1) { if ($i == 1) { continue; } if ($i == 4) { break; } echo $i; }"
	got := run(t, src)
	if got != "023" {
		t.Fatalf("got %q", got)
	}
}

func TestDoWhileRunsAtLeastOnce(t *testing.T) {
	src := "<?php $i = 0; do { echo $i; $i = $i + 1; } while ($i < 3);"
	got := run(t, src)
	if got != "012" {
		t.Fatalf("got %q", got)
	}
}

func TestWhileBreakAndForeachContinue(t *testing.T) {
	src := "<?php $i = 0; while (1) { $i = $i + 1; if ($i == 2) { continue; } if ($i == 4) { break; } echo $i; } foreach (array(1, 2, 3) as $v) { if ($v == 2) { continue; } echo $v; }"
	got := run(t, src)
	if got != "1313" {
		t.Fatalf("got %q", got)
	}
}

func TestSwitchCaseDefaultAndFallthrough(t *testing.T) {
	src := "<?php $x = 2; switch ($x) { case 1: echo 'a'; break; case 2: echo 'b'; case 3: echo 'c'; break; default: echo 'd'; } switch (9) { case 1: echo 'x'; break; default: echo 'z'; }"
	got := run(t, src)
	if got != "bcz" {
		t.Fatalf("got %q", got)
	}
}

func TestNumericBreakAndContinue(t *testing.T) {
	src := "<?php for ($i = 0; $i < 3; $i = $i + 1) { for ($j = 0; $j < 3; $j = $j + 1) { if ($i == 1 && $j == 0) { continue 2; } if ($i == 2 && $j == 1) { break 2; } echo $i; echo $j; } } echo ':'; for ($k = 0; $k < 2; $k = $k + 1) { switch ($k) { case 1: continue 2; default: echo $k; } echo '-'; }"
	got := run(t, src)
	if got != "00010220:0-" {
		t.Fatalf("got %q", got)
	}
}

func TestNumericBreakAndContinueRejectExpressions(t *testing.T) {
	if errs := parseErrors("<?php while (1) { break 1 + 1; }"); len(errs) == 0 {
		t.Fatal("expected parse error for break expression level")
	}
	if errs := parseErrors("<?php while (1) { continue 1 + 1; }"); len(errs) == 0 {
		t.Fatal("expected parse error for continue expression level")
	}
}

func TestFunctionDefaultParameters(t *testing.T) {
	src := "<?php function label($name = 'php', $suffix = '!') { return $name . $suffix; } echo label(); echo label('go', '?');"
	got := run(t, src)
	if got != "php!go?" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionCompatibilityValidation(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{
			src:  "<?php function bad($optional = 1, $required) {}",
			want: "function bad required parameter $required cannot follow optional parameter $optional",
		},
		{
			src:  "<?php $x = 1; function bad_default($value = $x) {}",
			want: "function bad_default parameter $value default must be a constant expression",
		},
		{
			src:  "<?php $x = 1; $fn = function($value = $x) { return $value; };",
			want: "closure parameter $value default must be a constant expression",
		},
	}
	for _, tc := range cases {
		err := runErr(t, tc.src)
		if err == nil || err.Error() != tc.want {
			t.Fatalf("for %q got %v, want %q", tc.src, err, tc.want)
		}
	}
}

func TestFunctionCallReferenceAndNameRules(t *testing.T) {
	src := "<?php function Bump(&$value) { $value = $value + 1; } $items = array('n' => 1); bump($items['n']); echo $items['n']; $name = 'x'; $x = 2; BUMP($$name); echo $x;"
	got := run(t, src)
	if got != "23" {
		t.Fatalf("got %q", got)
	}

	err := runErr(t, "<?php function Same() {} function same() {}")
	if err == nil || err.Error() != "cannot redeclare function same" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalStatementSharesGlobalVariable(t *testing.T) {
	src := "<?php $x = 1; function bump() { global $x; $x = $x + 2; } bump(); echo $x;"
	got := run(t, src)
	if got != "3" {
		t.Fatalf("got %q", got)
	}
}

func TestStaticLocalPersistsAcrossCalls(t *testing.T) {
	src := "<?php function next_value() { static $i = 0; $i = $i + 1; return $i; } echo next_value(); echo next_value(); echo next_value();"
	got := run(t, src)
	if got != "123" {
		t.Fatalf("got %q", got)
	}
}

func TestPassByReferenceParameter(t *testing.T) {
	src := "<?php function inc(&$value) { $value = $value + 1; } $x = 4; inc($x); echo $x;"
	got := run(t, src)
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func TestPassByReferenceRequiresVariable(t *testing.T) {
	err := runErr(t, "<?php function inc(&$value) { $value = $value + 1; } inc(4);")
	if err == nil || err.Error() != "argument 1 for inc must be a variable passed by reference" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArrayElementReferences(t *testing.T) {
	src := "<?php $x = 1; $a = array(); $a[0] =& $x; $a[0] = 2; echo $x; $x = 3; echo $a[0]; $y =& $a[0]; $y = 4; echo $x; echo $a[0];"
	got := run(t, src)
	if got != "2344" {
		t.Fatalf("got %q", got)
	}
}

func TestForeachByReference(t *testing.T) {
	src := "<?php $items = array(1, 2); foreach ($items as &$value) { $value = $value + 1; } echo $items[0]; echo $items[1]; $value = 9; echo $items[1];"
	got := run(t, src)
	if got != "239" {
		t.Fatalf("got %q", got)
	}
}

func TestReturnByReference(t *testing.T) {
	src := "<?php function &counter_ref() { static $value = 1; return $value; } $ref =& counter_ref(); $ref = 7; echo counter_ref();"
	got := run(t, src)
	if got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestClosureUseCaptures(t *testing.T) {
	src := "<?php $base = 2; $add = function($value) use ($base) { return $base + $value; }; $base = 10; echo $add(3); $n = 1; $bump = function() use (&$n) { $n = $n + 1; return $n; }; echo $bump(); echo $n;"
	got := run(t, src)
	if got != "522" {
		t.Fatalf("got %q", got)
	}
}

func TestBuiltinIssetAndEmpty(t *testing.T) {
	src := "<?php $items = array('a' => 1, 'b' => null, 'c' => 'x'); if (isset($items['a'])) { echo 'a'; } if (!isset($items['b'])) { echo 'b'; } if (!isset($missing)) { echo 'c'; } if (empty($items['b'])) { echo 'd'; } if (empty(0)) { echo 'e'; } if (!empty($items['c'])) { echo 'f'; }"
	got := run(t, src)
	if got != "abcdef" {
		t.Fatalf("got %q", got)
	}
}

func TestBuiltinStringHelpers(t *testing.T) {
	src := "<?php echo strlen('abcdef'); echo ':'; echo substr('abcdef', 1, 3); echo ':'; echo substr('abcdef', -2); echo ':'; echo strpos('abcdef', 'de'); echo ':'; echo strtoupper('php'); echo ':'; echo strtolower('GO');"
	got := run(t, src)
	if got != "6:bcd:ef:3:PHP:go" {
		t.Fatalf("got %q", got)
	}
}

func TestBuiltinArrayHelpers(t *testing.T) {
	src := "<?php $items = array('a' => 'x', 'b' => 'y'); echo count($items); echo ':'; echo array_key_exists('b', $items); echo ':'; echo in_array('x', $items); echo ':'; echo implode('-', array('a', 'b')); echo ':'; $parts = explode(',', 'x,y,z'); echo count($parts); echo ':'; echo $parts[1];"
	got := run(t, src)
	if got != "2:1:1:a-b:3:y" {
		t.Fatalf("got %q", got)
	}
}

func TestBuiltinTypeChecks(t *testing.T) {
	src := "<?php if (is_array(array(1))) { echo 'a'; } if (is_string('x')) { echo 'b'; } if (is_null($missing)) { echo 'c'; }"
	got := run(t, src)
	if got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestMoreBuiltins(t *testing.T) {
	src := "<?php echo trim(' php '); echo ':'; echo str_replace('5.3', 'go', 'php 5.3'); echo ':'; echo round(1.6); echo ':'; echo floor(1.9); echo ':'; echo ceil(1.1); echo ':'; echo abs(-4); echo ':'; echo min(3, 1, 2); echo ':'; echo max(array(3, 7, 2)); echo ':'; echo date('Y-m-d', 0);"
	got := run(t, src)
	if got != "php:php go:2:1:2:4:1:7:1970-01-01" {
		t.Fatalf("got %q", got)
	}
}

func TestOutputBuiltinsFromInterpreter(t *testing.T) {
	src := "<?php print_r(array('name' => 'php')); echo '|'; var_dump('go'); echo '|'; echo print_r(array('x'), true);"
	got := run(t, src)
	want := "Array\n(\n    [name] => php\n)\n|string(2) \"go\"\n|Array\n(\n    [0] => x\n)\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestHeaderAndHeadersList(t *testing.T) {
	src := "<?php header('X-Test: one'); header('X-Test: two'); header('X-Trace: a', false); foreach (headers_list() as $header) { echo $header; echo '|'; }"
	got := run(t, src)
	if got != "X-Test: two|X-Trace: a|" {
		t.Fatalf("got %q", got)
	}
}

func TestFilesystemRegexAndSerializationBuiltins(t *testing.T) {
	src := "<?php echo basename('tests/phpt/basic.phpt'); echo ':'; echo basename(dirname('tests/phpt/basic.phpt')); echo ':'; echo file_exists('interpreter_test.go'); echo ':'; echo preg_match('/php/', 'php-src'); echo ':'; echo preg_replace('/5[.]3/', 'go', 'php 5.3'); echo ':'; $data = array('name' => 'php', 'version' => 53); $encoded = serialize($data); echo $encoded; echo ':'; $copy = unserialize($encoded); echo $copy['name']; echo ':'; echo $copy['version'];"
	got := run(t, src)
	want := `basic.phpt:phpt:1:1:php go:a:2:{s:4:"name";s:3:"php";s:7:"version";i:53;}:php:53`
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestPregMatchCapturesJSONHashAndFileContents(t *testing.T) {
	dir := t.TempDir()
	file := filepath.ToSlash(filepath.Join(dir, "data.txt"))
	src := "<?php echo file_put_contents('" + file + "', 'php'); echo ':'; echo file_get_contents('" + file + "'); echo ':'; echo preg_match('/(php)-([0-9]+)/', 'php-53', $matches); echo ':'; echo $matches[0]; echo ':'; echo $matches[1]; echo ':'; echo $matches[2]; echo ':'; $json = json_encode(array('name' => 'php', 'version' => 53)); echo $json; echo ':'; $copy = json_decode($json, true); echo $copy['name']; echo ':'; echo $copy['version']; echo ':'; echo md5('hello'); echo ':'; echo hash('sha1', 'hello');"
	got := run(t, src)
	want := `3:php:1:php-53:php:53:{"name":"php","version":53}:php:53:5d41402abc4b2a76b9719d911017c592:aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d`
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestP0SyntaxBatch(t *testing.T) {
	src := "<?php // line comment\n/* block comment */ $a = 5; $a += 3; $a %= 5; $a++; ++$a; $a -= 1; $a *= 3; $a /= 2; echo $a; echo ':'; $s = 'php'; $s .= '-go'; print $s; echo ':'; if ($a === 6.0) { echo 'strict'; } elseif ($a <> 6) { echo 'bad'; } else { echo 'bad'; } $items = array('x' => 1, 'y' => 2); unset($items['x']); if (!isset($items['x'])) { echo ':unset'; } ?>"
	got := run(t, src)
	want := "6:php-go:strict:unset"
	if got != want {
		t.Fatalf("got %q", got)
	}
}
