--TEST--
P2 is_callable for OOP callables
--FILE--
<?php
function callable_fn() {}

class CallableTarget {
    public function run() {}
    private function hidden() {}
    public static function stat() {}
}

class CallableMagic {
    public function __call($name, $args) {}
    public static function __callStatic($name, $args) {}
}

class_alias('CallableTarget', 'CallableAlias');

$obj = new CallableTarget();
$magic = new CallableMagic();
$fn = function() {};

echo is_callable('callable_fn') ? 'fn' : 'no';
echo ':';
echo is_callable($fn) ? 'closure' : 'no';
echo ':';
echo is_callable(array($obj, 'run')) ? 'method' : 'no';
echo ':';
echo is_callable(array($obj, 'hidden')) ? 'hidden' : 'no';
echo ':';
echo is_callable(array($magic, 'missing')) ? 'magic' : 'no';
echo ':';
echo is_callable(array('CallableTarget', 'stat'), false, $name) ? $name : 'no';
echo ':';
echo is_callable('CallableAlias::stat') ? 'alias' : 'no';
echo ':';
echo is_callable(array('CallableMagic', 'missing')) ? 'static-magic' : 'no';
echo ':';
echo is_callable(array('Missing', 'missing'), true) ? 'syntax' : 'no';
echo ':';
echo is_callable(array($obj, '__clone')) ? 'clone' : 'no';
--EXPECT--
fn:closure:method:no:magic:CallableTarget::stat:alias:static-magic:syntax:no
