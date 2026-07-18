--TEST--
P2 class cannot extend an interface
--FILE--
<?php
interface ExtendsInterfaceContract {}

class ExtendsInterfaceUser extends ExtendsInterfaceContract {}
?>
--EXPECTF--
class ExtendsInterfaceUser cannot extend interface ExtendsInterfaceContract
