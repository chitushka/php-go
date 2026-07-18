--TEST--
P2 class cannot be both final and abstract
--FILE--
<?php
abstract final class BadClassModifier {}
?>
--EXPECTF--
class BadClassModifier cannot be both final and abstract
