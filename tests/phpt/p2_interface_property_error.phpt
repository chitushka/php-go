--TEST--
P2 interfaces cannot include member variables
--FILE--
<?php
interface BadInterfaceProperty {
    public $value;
}
?>
--EXPECTF--
parse errors: [Interfaces may not include member variables]
