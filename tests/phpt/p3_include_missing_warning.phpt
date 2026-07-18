--TEST--
P3 missing include returns false and continues
--FILE--
<?php
$value = include 'p3_include_missing_warning_no_such_file.php';
echo ($value === false ? 'false' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
false
after
