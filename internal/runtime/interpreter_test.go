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

func TestMagicToString(t *testing.T) {
	src := "<?php class Label { public function __toString() { return 'label'; } } $label = new Label(); echo $label;"
	got := run(t, src)
	if got != "label" {
		t.Fatalf("got %q", got)
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

func TestFunctionDefaultParameters(t *testing.T) {
	src := "<?php function label($name = 'php', $suffix = '!') { return $name . $suffix; } echo label(); echo label('go', '?');"
	got := run(t, src)
	if got != "php!go?" {
		t.Fatalf("got %q", got)
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
