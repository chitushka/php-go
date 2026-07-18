--TEST--
P2 class constants cannot have modifiers
--FILE--
<?php
class BadClassConstModifier {
    public const VALUE = 1;
}
?>
--EXPECTF--
parse errors: [class constants cannot have modifiers]
