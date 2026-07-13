--TEST--
io json hash regex capture builtins
--FILE--
<?php
echo strpos(file_get_contents('tests/phpt/basic.phpt'), 'basic arithmetic');
echo ':';
echo preg_match('/(php)-([0-9]+)/', 'php-53', $matches);
echo ':';
echo $matches[0];
echo ':';
echo $matches[1];
echo ':';
echo $matches[2];
echo ':';
$json = json_encode(array('name' => 'php', 'version' => 53));
echo $json;
echo ':';
$copy = json_decode($json, true);
echo $copy['name'];
echo ':';
echo $copy['version'];
echo ':';
echo md5('hello');
echo ':';
echo sha1('hello');
--EXPECT--
9:1:php-53:php:53:{"name":"php","version":53}:php:53:5d41402abc4b2a76b9719d911017c592:aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d
