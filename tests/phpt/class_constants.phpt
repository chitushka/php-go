--TEST--
class constants
--FILE--
<?php
class Base {
    const VALUE = 4;
}
class Child extends Base {
    const LOCAL = 2;
    public function total() {
        return parent::VALUE + self::LOCAL;
    }
}
$child = new Child();
echo Child::VALUE;
echo ':';
echo $child->total();
--EXPECT--
4:6
