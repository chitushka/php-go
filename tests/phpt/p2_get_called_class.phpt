--TEST--
P2 get_called_class
--FILE--
<?php
class Base {
    public static function who() {
        return get_called_class();
    }

    public function instanceWho() {
        return get_called_class();
    }
}

class Child extends Base {
}

$child = new Child();
echo Base::who();
echo ':';
echo Child::who();
echo ':';
echo $child->instanceWho();
--EXPECT--
Base:Child:Child
