--TEST--
P2 __callStatic must be static
--FILE--
<?php
class NonStaticCallStaticMagic {
    public function __callStatic($name, $args) {}
}
?>
--EXPECTF--
magic method NonStaticCallStaticMagic::__callStatic must be static
