--TEST--
P3 registry builtin argument warnings are non-halting by default
--FILE--
<?php
$value = strlen();
echo ($value === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
