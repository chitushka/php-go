--TEST--
P3 OOP introspection warnings are non-halting by default
--FILE--
<?php
$a = get_class(1);
$b = get_object_vars(1);
$c = get_called_class();
$d = method_exists('Box');
echo ($a === null ? 'null' : 'bad');
echo "\n";
echo ($b === null ? 'null' : 'bad');
echo "\n";
echo ($c === null ? 'null' : 'bad');
echo "\n";
echo ($d === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
null
null
after
