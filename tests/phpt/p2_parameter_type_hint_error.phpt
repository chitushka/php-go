--TEST--
P2 parameter type hints reject incompatible runtime values
--FILE--
<?php
function needs_array(array $items) {}
needs_array('no');
?>
--EXPECTF--
argument 1 for needs_array must be of type array
