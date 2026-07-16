--TEST--
P2 property defaults must be constant expressions
--FILE--
<?php
$runtime = 1;

class BadPropertyDefault {
    public $value = $runtime;
}
?>
--EXPECTF--
property BadPropertyDefault::$value default must be a constant expression
