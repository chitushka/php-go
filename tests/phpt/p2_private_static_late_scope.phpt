--TEST--
P2 private static late binding uses declaring class scope
--FILE--
<?php
class LateBaseScope {
    private static $value = 'base';

    public static function viaStatic() {
        return static::$value;
    }

    public static function setViaStatic($value) {
        static::$value = $value;
    }
}

class LateChildScope extends LateBaseScope {
    private static $value = 'child';

    public static function childValue() {
        return self::$value;
    }
}

echo LateChildScope::viaStatic();
echo ':';
echo LateChildScope::childValue();
LateChildScope::setViaStatic('new-base');
echo ':';
echo LateBaseScope::viaStatic();
echo ':';
echo LateChildScope::viaStatic();
echo ':';
echo LateChildScope::childValue();
?>
--EXPECT--
base:child:new-base:new-base:child
