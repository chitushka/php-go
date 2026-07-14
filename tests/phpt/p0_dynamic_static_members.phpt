--TEST--
P0 dynamic static property and method names
--FILE--
<?php
class DynamicStatic {
    public static $value = 'seed';

    public static function show($suffix) {
        return self::$value . $suffix;
    }
}

$name = 'slot';
$slot = 'value';
echo DynamicStatic::$$name;
echo "\n";

DynamicStatic::$$name = 'changed';
echo DynamicStatic::$value;
echo "\n";

$method = 'show';
echo DynamicStatic::$method('!');
?>
--EXPECT--
seed
changed
changed!
