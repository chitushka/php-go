--TEST--
Types 002: string offset access (curly brace legacy syntax), interpolation forms
--FILE--
<?php
$s = "Hello";
var_dump($s[0]);
var_dump($s{1}); // legacy curly-brace string offset syntax, valid in PHP 5.3
                 // (deprecated in 7.4, REMOVED in PHP 8.0 - do not run this file's
                 // FILE section against PHP 8+, it will be a parse error there)
$s[0] = 'J';
var_dump($s);

$name = "World";
echo "Hello, $name!\n";
echo "Hello, {$name}!\n";
$arr = array('key' => 'value');
echo "Value: {$arr['key']}\n";
echo "Value: $arr[key]\n"; // simple syntax: unquoted key inside double-quoted string

class Box { public $item = 'apple'; }
$box = new Box();
echo "Item: {$box->item}\n";
echo "Item: $box->item\n";
?>
--EXPECT--
string(1) "H"
string(1) "e"
string(5) "Jello"
Hello, World!
Hello, World!
Value: value
Value: value
Item: apple
Item: apple
