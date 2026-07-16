--TEST--
P2 interface cannot extend the same interface twice
--FILE--
<?php
interface DuplicateParentContract {}

interface DuplicateChildContract extends DuplicateParentContract, duplicateparentcontract {}
?>
--EXPECTF--
interface DuplicateChildContract cannot extend interface DuplicateParentContract more than once
