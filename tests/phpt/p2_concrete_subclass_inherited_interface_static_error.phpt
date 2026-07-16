--TEST--
P2 concrete subclass inherited interface static modifier must match
--FILE--
<?php
interface InheritedStaticNamed {
    public static function name();
}
abstract class InheritedStaticBase implements InheritedStaticNamed {
}
class InheritedStaticUser extends InheritedStaticBase {
    public function name() {}
}
?>
--EXPECTF--
method InheritedStaticUser::name static modifier must match InheritedStaticNamed::name
