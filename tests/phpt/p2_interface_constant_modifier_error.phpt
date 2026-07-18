--TEST--
P2 interface constants cannot have modifiers
--FILE--
<?php
interface BadInterfaceConstModifier {
    public const VALUE = 1;
}
?>
--EXPECTF--
parse errors: [interface constants cannot have modifiers]
