--TEST--
P2 method override signature compatibility
--FILE--
<?php
class Base {
    public function label($name) {
        return $name;
    }
}

class Child extends Base {
    public function label($name, $suffix = '!') {
        return $name . $suffix;
    }
}

$child = new Child();
echo $child->label('php');
echo "\n";
echo $child->label('go', '?');
?>
--EXPECT--
php!
go?
