--TEST--
P2 class cannot implement the same interface twice
--FILE--
<?php
interface DuplicateContract {}

class DuplicateUser implements DuplicateContract, duplicatecontract {}
?>
--EXPECTF--
class DuplicateUser cannot implement interface DuplicateContract more than once
