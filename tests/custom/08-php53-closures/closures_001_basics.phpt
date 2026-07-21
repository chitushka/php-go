--TEST--
Closures 001 (NEW in PHP 5.3): anonymous functions, use() by value, use() by reference, $this binding
--FILE--
<?php
$greeting = function ($name) {
    return "Hello, $name!";
};
var_dump($greeting('World'));
var_dump($greeting instanceof Closure);
var_dump(is_callable($greeting));

$x = 10;
$byValue = function () use ($x) {
    return $x;
};
$x = 20;
var_dump($byValue()); // captures value at definition time: 10

$y = 10;
$byRef = function () use (&$y) {
    return $y;
};
$y = 20;
var_dump($byRef()); // captures by reference: 20

function makeAdder($n) {
    return function ($x) use ($n) {
        return $x + $n;
    };
}
$add5 = makeAdder(5);
$add10 = makeAdder(10);
var_dump($add5(1));
var_dump($add10(1));

function makeCounter() {
    $count = 0;
    return function () use (&$count) {
        $count++;
        return $count;
    };
}
$counter = makeCounter();
var_dump($counter());
var_dump($counter());
var_dump($counter());

class Multiplier {
    private $factor;
    public function __construct($factor) { $this->factor = $factor; }
    public function getClosure() {
        // In PHP 5.3, a closure defined inside a method automatically
        // binds $this if $this is used inside the closure body.
        return function ($x) {
            return $x * $this->factor;
        };
    }
}
$m = new Multiplier(3);
$triple = $m->getClosure();
var_dump($triple(7));

$funcs = array();
for ($i = 0; $i < 3; $i++) {
    $funcs[] = function () use ($i) { return $i; };
}
var_dump($funcs[0]());
var_dump($funcs[1]());
var_dump($funcs[2]());
?>
--EXPECT--
string(13) "Hello, World!"
bool(true)
bool(true)
int(10)
int(20)
int(6)
int(11)
int(1)
int(2)
int(3)
int(21)
int(0)
int(1)
int(2)
