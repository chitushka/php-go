--TEST--
P0 dynamic object property and method names
--FILE--
<?php
class Bag {
    public $value = 'seed';

    public function show($suffix) {
        return $this->value . $suffix;
    }

    public function __call($name, $args) {
        return 'magic-' . $name . '-' . $args[0];
    }
}

$bag = new Bag();
$prop = 'value';
echo $bag->$prop;
echo "\n";

$bag->$prop = 'changed';
echo $bag->$prop;
echo "\n";

$method = 'show';
echo $bag->$method('!');
echo "\n";

$missing = 'run';
echo $bag->$missing('x');
?>
--EXPECT--
seed
changed
changed!
magic-run-x
