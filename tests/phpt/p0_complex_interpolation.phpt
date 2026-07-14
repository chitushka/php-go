--TEST--
P0 complex double quoted interpolation
--FILE--
<?php
class User {
    public $name = 'Ada';
}

$name = 'PHP';
$holder = 'name';
$items = array('lang' => 'Go', 2 => 'two');
$user = new User();

echo "simple array $items[lang]";
echo "\n";
echo "numeric array $items[2]";
echo "\n";
echo "object $user->name";
echo "\n";
echo "dollar brace ${name}";
echo "\n";
echo "variable variable ${$holder}";
echo "\n";
echo "complex {$items['lang']} {$user->name}";
?>
--EXPECT--
simple array Go
numeric array two
object Ada
dollar brace PHP
variable variable PHP
complex Go Ada
