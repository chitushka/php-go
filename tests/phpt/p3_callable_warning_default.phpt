--TEST--
P3 callable builtin warnings are non-halting by default
--FILE--
<?php
$a = call_user_func();
$b = call_user_func_array('strlen', 'not-array');
$c = call_user_func(array('not-enough'));
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
