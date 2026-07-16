--TEST--
P2 abstract class interface method must be public when declared
--FILE--
<?php
interface AbstractHiddenNamed {
    public function name();
}

abstract class AbstractHiddenUser implements AbstractHiddenNamed {
    protected function name() {}
}
?>
--EXPECTF--
class AbstractHiddenUser interface method name must be public
