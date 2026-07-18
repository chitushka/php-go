--TEST--
P3 isset and empty argument warnings are non-halting by default
--FILE--
<?php
$a = isset();
$b = empty();
$c = empty($a, $b);
echo ($a === null ? 'null' : 'bad');
echo "\n";
echo ($b === null ? 'null' : 'bad');
echo "\n";
echo ($c === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
null
after
