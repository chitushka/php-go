--TEST--
P2 private static parent members use declaring class scope
--FILE--
<?php
class Base {
    private static $value = 'base';

    public static function baseValue() {
        return self::$value;
    }

    public static function setBase($value) {
        self::$value = $value;
    }
}

class Child extends Base {
    private static $value = 'child';

    public static function childValue() {
        return self::$value;
    }

    public static function setChild($value) {
        self::$value = $value;
    }
}

echo Base::baseValue();
echo ':';
echo Child::baseValue();
echo ':';
echo Child::childValue();

Child::setChild('new-child');
Base::setBase('new-base');

echo ':';
echo Base::baseValue();
echo ':';
echo Child::baseValue();
echo ':';
echo Child::childValue();
?>
--EXPECT--
base:base:child:new-base:new-base:new-child
