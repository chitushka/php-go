--TEST--
Functions 001: variable functions, callbacks, recursion, references, type-hinted args
--FILE--
<?php
function greet($name) { return "Hi, $name"; }
$fn = 'greet';
var_dump($fn('Ann'));

function apply($callback, $value) {
    return $callback($value);
}
var_dump(apply('strtoupper', 'abc'));
var_dump(apply(function ($x) { return $x * 2; }, 21));

function fact($n) {
    if ($n <= 1) return 1;
    return $n * fact($n - 1);
}
var_dump(fact(5));

function fillArray(array $arr) {
    return count($arr);
}
var_dump(fillArray(array(1, 2, 3)));

class Greeter {
    public function hello($name) { return "Hello $name"; }
    public static function shout($name) { return strtoupper("Hello $name"); }
}
$g = new Greeter();
var_dump(call_user_func(array($g, 'hello'), 'Bob'));
var_dump(call_user_func(array('Greeter', 'shout'), 'Bob'));
var_dump(call_user_func('Greeter::shout', 'Sam'));

function swap(&$a, &$b) {
    $tmp = $a;
    $a = $b;
    $b = $tmp;
}
$x = 1; $y = 2;
swap($x, $y);
var_dump($x, $y);
?>
--EXPECT--
string(7) "Hi, Ann"
string(3) "ABC"
int(42)
int(120)
int(3)
string(9) "Hello Bob"
string(9) "HELLO BOB"
string(9) "HELLO SAM"
int(2)
int(1)
