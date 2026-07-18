--TEST--
P2 interface cannot extend a class
--FILE--
<?php
class InterfaceExtendsClassBase {}

interface InterfaceExtendsClassUser extends InterfaceExtendsClassBase {}
?>
--EXPECTF--
interface InterfaceExtendsClassUser cannot extend class InterfaceExtendsClassBase
