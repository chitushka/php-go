--TEST--
Namespaces 001 (NEW in PHP 5.3): declaration, __NAMESPACE__, fully-qualified names, use/aliasing
--FILE--
<?php
namespace App\Models;

const VERSION = '1.0';

function helper() {
    return 'helper-in-' . __NAMESPACE__;
}

class User {
    public $name = 'anon';
    public function whoAmI() {
        return __CLASS__;
    }
}

var_dump(__NAMESPACE__);
var_dump(helper());
var_dump(VERSION);

$u = new User();
var_dump($u->whoAmI());
var_dump(get_class($u));

namespace App\Services;

use App\Models\User as ModelUser;

class UserService {
    public function build() {
        return new ModelUser();
    }
    public function buildFullyQualified() {
        return new \App\Models\User();
    }
}

$service = new UserService();
$built = $service->build();
var_dump(get_class($built));

$built2 = $service->buildFullyQualified();
var_dump(get_class($built2));

var_dump(\App\Models\VERSION);
var_dump(\App\Models\helper());

var_dump(function_exists('App\Models\helper'));
var_dump(class_exists('App\Models\User'));
var_dump(class_exists('App\Services\UserService'));
?>
--EXPECT--
string(10) "App\Models"
string(20) "helper-in-App\Models"
string(3) "1.0"
string(15) "App\Models\User"
string(15) "App\Models\User"
string(15) "App\Models\User"
string(15) "App\Models\User"
string(3) "1.0"
string(20) "helper-in-App\Models"
bool(true)
bool(true)
bool(true)
