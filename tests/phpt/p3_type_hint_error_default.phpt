--TEST--
P3 parameter type hint errors halt by default
--FILE--
<?php
function needs_array(array $items) {}
needs_array('no');
echo 'bad';
?>
--EXPECT--
argument 1 for needs_array must be of type array
--EXIT_CODE--
1
