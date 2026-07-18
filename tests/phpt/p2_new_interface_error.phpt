--TEST--
P2 interfaces cannot be instantiated
--FILE--
<?php
interface NewInterfaceContract {}

new NewInterfaceContract();
?>
--EXPECTF--
cannot instantiate interface NewInterfaceContract
