--TEST--
P3 undefined read notices are non-halting by default
--FILE--
<?php
$arr = array('yes' => 1);
$name = 'dynamicMissing';
$s = 'php';
class Bag {}
$bag = new Bag();
$a = $missing;
$b = $$name;
$c = $arr['no'];
$d = $s[9];
$e = $bag->item;
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
echo 'after';
?>
--EXPECT--
null
null
null
null
null
after
