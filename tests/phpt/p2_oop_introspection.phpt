--TEST--
P2 OOP introspection functions
--FILE--
<?php
function __autoload($class) {
    include 'p2_oop_introspection_lib.php';
}

echo class_exists('InspectTarget', false) ? 'loaded' : 'missing';
echo "\n";
echo interface_exists('InspectContract') ? 'iface' : 'no';
echo ':';
echo class_exists('inspecttarget') ? 'class' : 'no';
echo "\n";

$obj = new INSPECTTARGET();
$obj->dynamic = 1;
echo get_class($obj);
echo ':';
echo $obj->run();
echo "\n";

echo method_exists($obj, 'secret') ? 'secret' : 'no';
echo ':';
echo method_exists('InspectBase', 'hiddenMethod') ? 'hidden' : 'no';
echo "\n";

echo property_exists($obj, 'hidden') ? 'hidden' : 'no';
echo ':';
echo property_exists($obj, 'dynamic') ? 'dynamic' : 'no';
echo ':';
echo property_exists('InspectTarget', 'missing') ? 'missing' : 'no';
--EXPECT--
missing
iface:class
InspectTarget:InspectTarget
secret:hidden
hidden:dynamic:no
