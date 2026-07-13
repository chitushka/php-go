--TEST--
magic methods
--FILE--
<?php
class Magic {
    private $value = 'old';

    public function __set($name, $value) {
        $this->value = $value;
    }

    public function __get($name) {
        return $this->value;
    }

    public function __call($name, $args) {
        echo $name;
        foreach ($args as $arg) {
            echo ':';
            echo $arg;
        }
    }

    public function __toString() {
        return 'object';
    }
}

$magic = new Magic();
$magic->missing = 'stored';
echo $magic->missing;
echo '|';
$magic->run(1, 'x');
echo '|';
echo $magic;
--EXPECT--
stored|run:1:x|object
