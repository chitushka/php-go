--TEST--
extends visibility and static members
--FILE--
<?php
class Base {
    public $value = 3;

    protected function getValue() {
        return $this->value;
    }
}

class Counter extends Base {
    public static $count = 0;

    public static function inc() {
        Counter::$count = Counter::$count + 1;
        return Counter::$count;
    }

    public function double() {
        return $this->getValue() * 2;
    }
}

$counter = new Counter();
echo $counter->double();
echo Counter::inc();
echo Counter::inc();
--EXPECT--
612
