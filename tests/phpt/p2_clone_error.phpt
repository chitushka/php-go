--TEST--
P2 clone requires object
--FILE--
<?php
$x = clone 1;
?>
--EXPECTF--
clone expects object
