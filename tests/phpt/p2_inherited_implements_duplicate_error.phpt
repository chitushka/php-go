--TEST--
P2 class cannot re-implement parent interface
--FILE--
<?php
interface ParentContract {}

class ParentUser implements ParentContract {}

class ChildUser extends ParentUser implements ParentContract {}
?>
--EXPECTF--
class ChildUser cannot implement already implemented interface ParentContract
