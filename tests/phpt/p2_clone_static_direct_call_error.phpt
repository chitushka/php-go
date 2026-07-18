--TEST--
P2 __clone cannot be called statically
--FILE--
<?php
class StaticCloneCall {
    public function __clone() {}
}

StaticCloneCall::__clone();
?>
--EXPECTF--
cannot call __clone directly; use clone
