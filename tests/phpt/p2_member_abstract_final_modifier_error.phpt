--TEST--
P2 class members cannot be both abstract and final
--FILE--
<?php
abstract class BadAbstractFinalModifier {
    abstract final function value();
}
?>
--EXPECTF--
parse errors: [Cannot use the final modifier on an abstract class member]
