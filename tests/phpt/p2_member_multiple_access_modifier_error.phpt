--TEST--
P2 class members cannot have multiple access modifiers
--FILE--
<?php
class BadAccessModifier {
    public private function value() {}
}
?>
--EXPECTF--
parse errors: [Multiple access type modifiers are not allowed]
