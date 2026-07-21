--TEST--
P2 unset construct writable targets
--FILE--
<?php
class UnsetBase {
    public static $base = 'base';
}

class UnsetTarget extends UnsetBase {
    public static $count = 7;
    public $name = 'Ada';
}

$var = 'v';
$items = array('x' => 1, 'y' => 2);
$obj = new UnsetTarget();
$prop = 'name';
$static = 'count';

unset($var, $items['x'], $obj->$prop, UnsetTarget::$$static);

echo isset($var) ? 'var' : 'no-var';
echo ':';
echo isset($items['x']) ? 'x' : 'no-x';
echo ':';
echo isset($obj->name) ? 'name' : 'no-name';
echo ':';
echo isset(UnsetTarget::$count) ? 'count' : 'no-count';

unset(UnsetTarget::$base);
echo ':';
echo isset(UnsetBase::$base) ? 'base' : 'no-base';
--EXPECT--
no-var:no-x:no-name:no-count:no-base
