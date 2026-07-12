--TEST--
private property access is denied
--FILE--
<?php
class Sample {
    private $value = 1;
}

$sample = new Sample();
echo $sample->value;
--EXPECTF--
cannot access private property Sample::$value
