--TEST--
P3 constant and serialization warnings are non-halting by default
--FILE--
<?php
$a = define();
$b = defined();
$c = constant();
$d = constant('NO_SUCH_CONST');
$e = serialize();
$f = unserialize('broken');
echo ($a === null ? 'null' : 'bad');
echo "\n";
echo ($b === null ? 'null' : 'bad');
echo "\n";
echo ($c === null ? 'null' : 'bad');
echo "\n";
echo ($d === null ? 'null' : 'bad');
echo "\n";
echo ($e === null ? 'null' : 'bad');
echo "\n";
echo ($f === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
null
null
null
false
after
