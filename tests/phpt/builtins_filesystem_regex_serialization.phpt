--TEST--
filesystem regex serialization builtins
--FILE--
<?php
echo basename('tests/phpt/basic.phpt');
echo ':';
echo file_exists('tests/phpt/basic.phpt');
echo ':';
echo is_file('tests/phpt/basic.phpt');
echo ':';
echo is_dir('tests/phpt');
echo ':';
echo preg_match('/php/', 'php-src');
echo ':';
echo preg_replace('/5[.]3/', 'go', 'php 5.3');
echo ':';
$data = array('name' => 'php', 'version' => 53);
$encoded = serialize($data);
echo $encoded;
echo ':';
$copy = unserialize($encoded);
echo $copy['name'];
echo ':';
echo $copy['version'];
--EXPECT--
basic.phpt:1:1:1:1:php go:a:2:{s:4:"name";s:3:"php";s:7:"version";i:53;}:php:53
