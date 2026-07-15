--TEST--
P2 clone and __clone
--FILE--
<?php
class Box {
    public $value = 'old';

    public function __clone() {
        $this->value = 'clone';
        echo "cloned\n";
    }
}

$a = new Box();
$b = clone $a;
$b->extra = 'copy';
echo $a->value . "\n";
echo $b->value . ':' . $b->extra . "\n";
?>
--EXPECT--
cloned
old
clone:copy
