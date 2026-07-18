--TEST--
P3 preg_match capture warnings are non-halting by default
--FILE--
<?php
$a = preg_match('/x/', 'x', 'not-writable');
$b = preg_match('/x/', 'x', $matches, 'extra');
$c = preg_match('/[/', 'x', $matches);
echo ($a === null ? 'null' : 'bad');
echo "\n";
echo ($b === null ? 'null' : 'bad');
echo "\n";
echo ($c === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
false
after
