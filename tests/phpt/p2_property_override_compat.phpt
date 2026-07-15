--TEST--
P2 inherited property compatibility
--FILE--
<?php
class Base {
    protected $value = 1;
    private $local = 'base';
}

class Child extends Base {
    public $value = 2;
    public $local = 'child';
}

$child = new Child();
echo $child->value;
echo "\n";
echo $child->local;
?>
--EXPECT--
2
child
