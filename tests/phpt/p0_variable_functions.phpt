--TEST--
P0 variable functions
--FILE--
<?php
function greet($name) {
    return 'hi ' . $name;
}

$fn = 'greet';
echo $fn('Ada');
echo "\n";

$builtin = 'strtoupper';
echo $builtin('php');
echo "\n";

$holder = 'fn';
echo $$holder('Bob');
?>
--EXPECT--
hi Ada
PHP
hi Bob
