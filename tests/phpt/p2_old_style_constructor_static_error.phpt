--TEST--
P2 old-style constructor must not be static
--FILE--
<?php
class StaticOldStyle {
    public static function StaticOldStyle() {}
}
?>
--EXPECTF--
old-style constructor StaticOldStyle::StaticOldStyle must not be static
