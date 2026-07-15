--TEST--
P2 private __clone can run from inside class scope
--FILE--
<?php
class Box {
    public $value = 'old';

    private function __clone() {
        $this->value = 'clone';
    }

    public function copy() {
        return clone $this;
    }
}

$a = new Box();
$b = $a->copy();
echo $a->value;
echo ':';
echo $b->value;
--EXPECT--
old:clone
