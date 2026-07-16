--TEST--
P2 instanceof supports dynamic class names
--FILE--
<?php
interface DynamicNamed {}

class DynamicBase {}

class DynamicUser extends DynamicBase implements DynamicNamed {}

$user = new DynamicUser();
$class = 'DynamicBase';
$interface = 'DynamicNamed';
$missing = 'DynamicMissing';
$holder = 'class';
$base = new DynamicBase();

echo $user instanceof $class;
echo ":";
echo $user instanceof $interface;
echo ":";
echo $user instanceof $missing;
echo ":";
echo $user instanceof $$holder;
echo ":";
echo $user instanceof $base;
echo ":";
echo $base instanceof $user;
?>
--EXPECT--
1:1::1:1:
