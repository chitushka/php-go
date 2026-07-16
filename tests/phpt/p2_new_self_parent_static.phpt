--TEST--
P2 new self parent and static class names
--FILE--
<?php
class BaseNewName {
    public $name = 'base';

    public function makeSelf() {
        return new self();
    }

    public static function makeStatic() {
        return new static();
    }
}

class ChildNewName extends BaseNewName {
    public $name = 'child';

    public function makeParent() {
        return new parent();
    }
}

$child = new ChildNewName();
echo get_class($child->makeSelf());
echo ":";
echo get_class($child->makeParent());
echo ":";
echo get_class(ChildNewName::makeStatic());
echo ":";
echo ChildNewName::makeStatic()->name;
?>
--EXPECT--
BaseNewName:BaseNewName:ChildNewName:child
