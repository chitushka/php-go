--TEST--
P2 class cannot implement a class
--FILE--
<?php
class ImplementsClassBase {}

class ImplementsClassUser implements ImplementsClassBase {}
?>
--EXPECTF--
class ImplementsClassUser cannot implement class ImplementsClassBase
