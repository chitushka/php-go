--TEST--
P2 __autoload hook loads classes
--FILE--
<?php
function __autoload($class) {
    echo 'autoload:';
    echo $class;
    echo "\n";
    include 'p2_autoload_lib.php';
}

echo autoloadedcase::LABEL;
echo "\n";

$obj = new AUTOLOADEDCASE();
echo $obj instanceof autoloadedcontract;
echo ':';
echo $obj->name();
--EXPECT--
autoload:autoloadedcase
class
1:loaded
