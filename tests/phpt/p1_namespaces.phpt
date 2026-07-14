--TEST--
P1 namespaces and imports
--FILE--
<?php
namespace Vendor\Lib;

define('GLOBAL_SUFFIX', '!');
define('Vendor\Lib\LOCAL_CONST', 'const');

function label($value) {
    return __NAMESPACE__ . ':' . $value . GLOBAL_SUFFIX;
}

class Box {
    const NAME = 'box';
    public function name() {
        return self::NAME;
    }
}

namespace App;
use Vendor\Lib\Box as ImportedBox;

function local() {
    return 'local';
}

$box = new ImportedBox();
echo \Vendor\Lib\label('go');
echo "\n";
echo local();
echo "\n";
echo strlen('abcd');
echo "\n";
echo $box->name();
echo "\n";
echo \Vendor\Lib\LOCAL_CONST;
echo "\n";
echo __NAMESPACE__;
?>
--EXPECT--
Vendor\Lib:go!
local
4
box
const
App
