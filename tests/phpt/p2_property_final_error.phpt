--TEST--
P2 properties cannot be final
--FILE--
<?php
class FinalPropertyTarget {
    final public $value;
}
?>
--EXPECTF--
property FinalPropertyTarget::$value cannot be final
