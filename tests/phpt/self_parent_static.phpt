--TEST--
self parent and static binding
--FILE--
<?php
class Base {
    public static $name = 'base';

    protected function value() {
        return 4;
    }

    public static function name() {
        return static::$name;
    }
}

class Child extends Base {
    public static $name = 'child';

    public function value2() {
        return parent::value() * 2;
    }
}

echo Child::name();
$child = new Child();
echo $child->value2();
--EXPECT--
child8
