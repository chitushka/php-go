--TEST--
more builtin functions
--FILE--
<?php
echo trim(' php ');
echo ':';
echo str_replace('5.3', 'go', 'php 5.3');
echo ':';
echo round(1.6);
echo ':';
echo floor(1.9);
echo ':';
echo ceil(1.1);
echo ':';
echo abs(-4);
echo ':';
echo min(3, 1, 2);
echo ':';
echo max(array(3, 7, 2));
echo ':';
echo date('Y-m-d', 0);
echo '
';
print_r(array('name' => 'php'));
var_dump('go');
--EXPECT--
php:php go:2:1:2:4:1:7:1970-01-01
Array
(
    [name] => php
)
string(2) "go"
