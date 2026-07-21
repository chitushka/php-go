--TEST--
P0 isset and empty require variable arguments
--FILE--
<?php
empty(0);
?>
--EXPECTF--
parse errors: [empty expects a variable]
