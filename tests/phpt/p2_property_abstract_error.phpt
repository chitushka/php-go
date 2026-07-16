--TEST--
P2 properties cannot be abstract
--FILE--
<?php
abstract class AbstractPropertyTarget {
    abstract public $value;
}
?>
--EXPECTF--
property AbstractPropertyTarget::$value cannot be abstract
