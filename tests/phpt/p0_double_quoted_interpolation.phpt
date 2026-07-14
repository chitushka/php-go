--TEST--
P0 double quoted interpolation
--FILE--
<?php
$name = 'PHP';
$version = 53;
$empty = null;
echo "Hello $name $version";
echo "\n";
echo "Missing $missing end";
echo "\n";
echo "Null:$empty";
echo "\n";
echo "Escaped \$name";
echo "\n";
echo 'Single $name';
?>
--EXPECT--
Hello PHP 53
Missing  end
Null:
Escaped $name
Single $name
