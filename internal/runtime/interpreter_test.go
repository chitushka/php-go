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

func TestClassConstructorMethodAndThis(t *testing.T) {
	src := "<?php class Box { function __construct($value) { $this->value = $value; } function get() { return $this->value; } } $box = new Box(9); echo $box->get();"
	got := run(t, src)
	if got != "9" {
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

func TestThrowAndCatchException(t *testing.T) {
	src := "<?php try { throw new Exception('boom'); } catch (Exception $e) { echo $e->getMessage(); } echo ' done';"
	got := run(t, src)
	if got != "boom done" {
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
